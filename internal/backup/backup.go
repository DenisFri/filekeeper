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
// It accepts a context for graceful shutdown support.
func RunBackup(ctx context.Context, cfg *config.Config, log *slog.Logger) error {

	pruneThreshold := time.Now().Add(-time.Duration(cfg.PruneAfterHours) * time.Hour)

	if cfg.EnableBackup {
		err := os.MkdirAll(cfg.BackupPath, os.ModePerm)
		if err != nil {
			return err
		}

		err = filepath.Walk(cfg.TargetFolder, func(path string, info os.FileInfo, err error) error {
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
				// Calculate relative path to preserve directory structure
				relPath, err := filepath.Rel(cfg.TargetFolder, path)
				if err != nil {
					return fmt.Errorf("failed to calculate relative path for %s: %w", path, err)
				}

				// Construct destination path preserving directory structure
				destPath := filepath.Join(cfg.BackupPath, relPath)

				// Create parent directories if they don't exist
				destDir := filepath.Dir(destPath)
				if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
					return fmt.Errorf("failed to create backup directory %s: %w", destDir, err)
				}

				startTime := time.Now()
				err = utils.CopyFile(path, destPath)
				if err != nil {
					return err
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
					err := utils.ExecuteRemoteCopy(destPath, cfg.RemoteBackup)
					if err != nil {
						return err
					}
					log.Info("copied to remote backup",
						slog.String("source", destPath),
						slog.String("remote", cfg.RemoteBackup),
						slog.Duration("duration", time.Since(remoteStart)),
					)
				}
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	// Check for cancellation before pruning
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Call function to prune old files
	err := pruner.PruneFiles(ctx, cfg.TargetFolder, pruneThreshold, log)
	if err != nil {
		return err
	}

	return nil
}
