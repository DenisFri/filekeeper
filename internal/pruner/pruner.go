package pruner

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PruneFiles removes files older than the pruneThreshold in the specified directory.
func PruneFiles(directory string, pruneThreshold time.Time) error {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file modification time is older than the prune threshold
		if !info.IsDir() && info.ModTime().Before(pruneThreshold) {
			err = os.Remove(path)
			if err != nil {
				return err
			}
			fmt.Printf("Pruned (deleted) %s\n", path)
		}

		return nil
	})

	return err
}
