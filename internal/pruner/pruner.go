package pruner

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func PruneFiles(directory string, pruneThreshold time.Time, log *slog.Logger) error {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
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
