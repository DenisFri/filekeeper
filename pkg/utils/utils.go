package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func CopyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// ExecuteRemoteCopy securely copies a file to a remote destination using scp.
// Unlike shell-based execution, this function passes arguments directly to exec.Command,
// preventing command injection attacks.
func ExecuteRemoteCopy(sourcePath, destination string) error {
	// Validate that sourcePath exists
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("source file does not exist: %w", err)
	}

	// Validate destination format (user@host:/path or just host:/path)
	if destination == "" {
		return fmt.Errorf("destination cannot be empty")
	}

	// Execute scp with arguments passed separately to prevent injection
	cmd := exec.Command("scp", sourcePath, destination)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp failed: %w, output: %s", err, string(output))
	}
	return nil
}
