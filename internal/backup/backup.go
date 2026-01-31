package backup

import (
	"filekeeper/internal/config"
	"filekeeper/internal/pruner"
	"filekeeper/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunBackup handles the backup and pruning of log files based on the PruneAfterHours configuration.
func RunBackup(cfg *config.Config) error {

	pruneThreshold := time.Now().Add(-time.Duration(cfg.PruneAfterHours) * time.Hour)

	if cfg.EnableBackup {
		err := os.MkdirAll(cfg.BackupPath, os.ModePerm)
		if err != nil {
			return err
		}

		err = filepath.Walk(cfg.TargetFolder, func(path string, info os.FileInfo, err error) error {
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

				err = utils.CopyFile(path, destPath)
				if err != nil {
					return err
				}
				fmt.Printf("Backed up %s to %s\n", path, destPath)

				// Optionally transfer the backup to a remote location
				if cfg.RemoteBackup != "" {
					err := utils.ExecuteRemoteCopy(destPath, cfg.RemoteBackup)
					if err != nil {
						return err
					}
					fmt.Printf("Copied %s to remote backup at %s\n", destPath, cfg.RemoteBackup)
				}
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	// Call function to prune old files
	err := pruner.PruneFiles(cfg.TargetFolder, pruneThreshold)
	if err != nil {
		return err
	}

	return nil
}
