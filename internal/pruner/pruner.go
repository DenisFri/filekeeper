package pruner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// PruneFiles deletes files older than pruneThreshold from the specified directory.
// It accepts a context for graceful shutdown support.
func PruneFiles(ctx context.Context, directory string, pruneThreshold time.Time, log *slog.Logger) error {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		// Check for context cancellation before processing each file
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		if !info.IsDir() && info.ModTime().Before(pruneThreshold) {
			err = os.Remove(path)
			if err != nil {
				return err
			}
			log.Info("pruned file",
				slog.String("path", path),
				slog.Int64("size_bytes", info.Size()),
				slog.Time("mod_time", info.ModTime()),
			)
		}

		return nil
	})

	return err
}
