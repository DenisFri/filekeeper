package backup

import (
	"context"
	"filekeeper/internal/config"
	"filekeeper/internal/pruner"
	"filekeeper/pkg/compression"
	"filekeeper/pkg/utils"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
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
		backupPaths := cfg.GetBackupPaths()

		// Create all backup directories
		for _, backupPath := range backupPaths {
			if err := os.MkdirAll(backupPath, os.ModePerm); err != nil {
				return result, fmt.Errorf("failed to create backup directory %s: %w", backupPath, err)
			}
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

			// Process file that needs backup to all destinations
			if err := backupFileToAllDestinations(ctx, path, info, cfg, opts, log, result); err != nil {
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

// backupFileToAllDestinations handles backing up a single file to all configured destinations.
// Local backups are performed in parallel, remote backups are performed sequentially.
// If compression is enabled, files are compressed during backup.
func backupFileToAllDestinations(ctx context.Context, path string, info os.FileInfo, cfg *config.Config, opts *RunOptions, log *slog.Logger, result *Result) error {
	// Calculate relative path to preserve directory structure
	relPath, err := filepath.Rel(cfg.TargetFolder, path)
	if err != nil {
		return fmt.Errorf("calculate relative path: %w", err)
	}

	backupPaths := cfg.GetBackupPaths()
	remoteBackups := cfg.GetRemoteBackups()
	compressionCfg := cfg.GetCompressionConfig()

	// In dry-run mode, just log what would happen
	if opts.DryRun {
		for _, backupPath := range backupPaths {
			destPath := filepath.Join(backupPath, relPath)
			finalPath := compression.GetDestinationPath(destPath, compressionCfg)
			log.Info("[DRY-RUN] would backup file",
				slog.String("source", path),
				slog.String("destination", finalPath),
				slog.Int64("size_bytes", info.Size()),
				slog.Bool("compressed", compressionCfg.Enabled),
			)
		}
		for _, remote := range remoteBackups {
			log.Info("[DRY-RUN] would copy to remote",
				slog.String("source", path),
				slog.String("remote", remote),
			)
		}
		return nil
	}

	// Backup to all local destinations in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(backupPaths))
	type backupResult struct {
		destPath       string
		compressResult *compression.Result
	}
	successChan := make(chan backupResult, len(backupPaths))

	for _, backupPath := range backupPaths {
		wg.Add(1)
		go func(bp string) {
			defer wg.Done()

			destPath := filepath.Join(bp, relPath)

			// Create parent directories if they don't exist
			destDir := filepath.Dir(destPath)
			if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
				errChan <- fmt.Errorf("create backup directory %s: %w", destDir, err)
				return
			}

			startTime := time.Now()

			// Use compression if enabled, otherwise do regular copy
			compResult, err := compression.CompressFile(path, destPath, compressionCfg)
			if err != nil {
				errChan <- fmt.Errorf("backup to %s: %w", bp, err)
				return
			}

			finalPath := compression.GetDestinationPath(destPath, compressionCfg)

			// Log with compression info if enabled
			if compressionCfg.Enabled && compResult.Algorithm != compression.None {
				log.Info("backed up file (compressed)",
					slog.String("source", path),
					slog.String("destination", finalPath),
					slog.Int64("original_bytes", compResult.OriginalSize),
					slog.Int64("compressed_bytes", compResult.CompressedSize),
					slog.Float64("compression_ratio", compResult.CompressionRatio()),
					slog.String("algorithm", string(compResult.Algorithm)),
					slog.Duration("duration", time.Since(startTime)),
				)
			} else {
				log.Info("backed up file",
					slog.String("source", path),
					slog.String("destination", finalPath),
					slog.Int64("size_bytes", info.Size()),
					slog.Duration("duration", time.Since(startTime)),
				)
			}
			successChan <- backupResult{destPath: finalPath, compressResult: compResult}
		}(backupPath)
	}

	// Wait for all local backups to complete
	wg.Wait()
	close(errChan)
	close(successChan)

	// Collect errors from local backups
	var localErrors []error
	for err := range errChan {
		localErrors = append(localErrors, err)
	}

	// Collect successful local backup results (for remote copy and compression stats)
	var successfulResults []backupResult
	for br := range successChan {
		successfulResults = append(successfulResults, br)

		// Track compression statistics
		if br.compressResult != nil && compressionCfg.Enabled {
			result.CompressedBytes += br.compressResult.CompressedSize
			result.OriginalBytes += br.compressResult.OriginalSize
		}
	}

	// If all local backups failed, return error
	if len(successfulResults) == 0 && len(backupPaths) > 0 {
		if len(localErrors) > 0 {
			return fmt.Errorf("all local backups failed: %v", localErrors[0])
		}
		return fmt.Errorf("all local backups failed")
	}

	// Log warnings for any failed local backups (but continue since at least one succeeded)
	for _, err := range localErrors {
		log.Warn("local backup failed",
			slog.String("path", path),
			slog.String("error", err.Error()),
		)
	}

	// Backup to remote destinations sequentially (to avoid bandwidth saturation)
	// Use the first successful local backup path as the source
	if len(remoteBackups) > 0 && len(successfulResults) > 0 {
		sourcePath := successfulResults[0].destPath

		for _, remote := range remoteBackups {
			// Check for cancellation before each remote copy
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			remoteStart := time.Now()
			if err := utils.ExecuteRemoteCopy(sourcePath, remote); err != nil {
				// Log warning but continue with other remote destinations
				log.Warn("remote backup failed",
					slog.String("source", sourcePath),
					slog.String("remote", remote),
					slog.String("error", err.Error()),
				)
				continue
			}

			log.Info("copied to remote backup",
				slog.String("source", sourcePath),
				slog.String("remote", remote),
				slog.Duration("duration", time.Since(remoteStart)),
			)
			result.RemoteCopied++
		}
	}

	return nil
}
