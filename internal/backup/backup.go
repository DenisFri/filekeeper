package backup

import (
	"context"
	"filekeeper/internal/config"
	"filekeeper/internal/pruner"
	"filekeeper/pkg/utils"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// RunBackup handles the backup and pruning of log files based on the PruneAfterHours configuration.
// It accepts a context for graceful shutdown support and returns a Result with success/failure counts.
// Individual file errors are logged but processing continues unless error threshold is exceeded.
// If opts.DryRun is true, it shows what would be done without making changes.
func RunBackup(ctx context.Context, cfg *config.Config, opts *RunOptions, log *slog.Logger) (*Result, error) {
	if opts == nil {
		opts = &RunOptions{}
	}
	result := NewResult()
	pruneThreshold := time.Now().Add(-time.Duration(cfg.PruneAfterHours) * time.Hour)

	if cfg.EnableBackup {
		if err := os.MkdirAll(cfg.BackupPath, os.ModePerm); err != nil {
			return result, fmt.Errorf("failed to create backup directory: %w", err)
		}

		err := filepath.Walk(cfg.TargetFolder, func(path string, info os.FileInfo, err error) error {
			// Check for context cancellation before processing each file
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Handle access errors - log and continue
			if err != nil {
				log.Warn("failed to access file",
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

			// Process file that needs backup
			if err := backupFile(ctx, path, info, cfg, opts, log, result); err != nil {
				// Check if this was a context cancellation
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// Log error but continue processing
				log.Error("backup failed",
					slog.String("path", path),
					slog.String("error", err.Error()),
				)
				result.AddError(path, "backup", err)

				// Check error threshold
				if cfg.ErrorThresholdPercent > 0 && result.FailureRate() > cfg.ErrorThresholdPercent {
					return fmt.Errorf("error threshold exceeded: %.1f%% failures (threshold: %.1f%%)",
						result.FailureRate(), cfg.ErrorThresholdPercent)
				}
				return nil // Continue walking
			}

			result.AddSuccess(info.Size())
			result.BackedUp++
			return nil
		})

		if err != nil {
			return result, err
		}
	}

	// Check for cancellation before pruning
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
	}

	// Call function to prune old files
	pruneResult, err := pruner.PruneFiles(ctx, cfg.TargetFolder, pruneThreshold, cfg.ErrorThresholdPercent, opts.DryRun, log)
	if pruneResult != nil {
		result.Pruned = pruneResult.Pruned
		result.Failed += pruneResult.Failed
		// Convert pruner errors to backup errors
		for _, e := range pruneResult.Errors {
			result.Errors = append(result.Errors, FileError{
				Path:      e.Path,
				Operation: e.Operation,
				Err:       e.Err,
			})
		}
	}
	if err != nil {
		return result, err
	}

	return result, nil
}

// backupFile handles backing up a single file to local and optionally remote destinations.
func backupFile(ctx context.Context, path string, info os.FileInfo, cfg *config.Config, opts *RunOptions, log *slog.Logger, result *Result) error {
	// Calculate relative path to preserve directory structure
	relPath, err := filepath.Rel(cfg.TargetFolder, path)
	if err != nil {
		return fmt.Errorf("calculate relative path: %w", err)
	}

	// Construct destination path preserving directory structure
	destPath := filepath.Join(cfg.BackupPath, relPath)

	// In dry-run mode, just log what would happen
	if opts.DryRun {
		log.Info("[DRY-RUN] would backup file",
			slog.String("source", path),
			slog.String("destination", destPath),
			slog.Int64("size_bytes", info.Size()),
		)
		if cfg.RemoteBackup != "" {
			log.Info("[DRY-RUN] would copy to remote",
				slog.String("source", destPath),
				slog.String("remote", cfg.RemoteBackup),
			)
		}
		return nil
	}

	// Create parent directories if they don't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("create backup directory %s: %w", destDir, err)
	}

	startTime := time.Now()
	if err := utils.CopyFile(path, destPath); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	log.Info("backed up file",
		slog.String("source", path),
		slog.String("destination", destPath),
		slog.Int64("size_bytes", info.Size()),
		slog.Duration("duration", time.Since(startTime)),
	)

	// Optionally transfer the backup to a remote location
	if cfg.RemoteBackup != "" {
		// Check for cancellation before remote copy
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		remoteStart := time.Now()
		if err := utils.ExecuteRemoteCopy(destPath, cfg.RemoteBackup); err != nil {
			return fmt.Errorf("remote copy to %s: %w", cfg.RemoteBackup, err)
		}
		log.Info("copied to remote backup",
			slog.String("source", destPath),
			slog.String("remote", cfg.RemoteBackup),
			slog.Duration("duration", time.Since(remoteStart)),
		)
		result.RemoteCopied++
	}

	return nil
}
