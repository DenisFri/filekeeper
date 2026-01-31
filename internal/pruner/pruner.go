package pruner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// PruneFiles deletes files older than pruneThreshold from the specified directory.
// It accepts a context for graceful shutdown support and returns a Result with success/failure counts.
// Individual file errors are logged but processing continues unless error threshold is exceeded.
func PruneFiles(ctx context.Context, directory string, pruneThreshold time.Time, errorThresholdPercent float64, log *slog.Logger) (*Result, error) {
	result := NewResult()

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		// Check for context cancellation before processing each file
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Handle access errors - log and continue
		if err != nil {
			log.Warn("failed to access file for pruning",
				slog.String("path", path),
				slog.String("error", err.Error()),
			)
			result.AddError(path, "access", err)
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil
		}

		if !info.ModTime().Before(pruneThreshold) {
			result.Skipped++
			return nil
		}

		// Attempt to remove the file
		if err := os.Remove(path); err != nil {
			log.Error("prune failed",
				slog.String("path", path),
				slog.String("error", err.Error()),
			)
			result.AddError(path, "prune", err)

			// Check error threshold
			if errorThresholdPercent > 0 && result.FailureRate() > errorThresholdPercent {
				return fmt.Errorf("error threshold exceeded: %.1f%% failures (threshold: %.1f%%)",
					result.FailureRate(), errorThresholdPercent)
			}
			return nil // Continue walking
		}

		log.Info("pruned file",
			slog.String("path", path),
			slog.Int64("size_bytes", info.Size()),
			slog.Time("mod_time", info.ModTime()),
		)
		result.Pruned++

		return nil
	})

	return result, err
}
