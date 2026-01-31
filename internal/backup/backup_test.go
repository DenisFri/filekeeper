package backup

import (
	"filekeeper/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRunBackup tests the RunBackup function
func TestRunBackup(t *testing.T) {
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp log directory: %v", err)
	}
	defer os.RemoveAll(logDir)

	backupDir, err := os.MkdirTemp("", "backupdir")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory: %v", err)
	}
	defer os.RemoveAll(backupDir)

	// Create a test log file that should be backed up and pruned
	oldFilePath := filepath.Join(logDir, "old.log")
	if err := os.WriteFile(oldFilePath, []byte("old log data"), 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	// Modify the file's modification time to be older than the prune threshold
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFilePath, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set modification time on old log file: %v", err)
	}

	// Create a test log file that should not be backed up or pruned
	newFilePath := filepath.Join(logDir, "new.log")
	if err := os.WriteFile(newFilePath, []byte("new log data"), 0644); err != nil {
		t.Fatalf("Failed to create new log file: %v", err)
	}

	// Define the config for the backup
	cfg := &config.Config{
		PruneAfterHours: 24, // Files older than 24 hours should be backed up and pruned
		BackupPath:      backupDir,
		EnableBackup:    true,
		TargetFolder:    logDir,
		RemoteBackup:    "",
	}

	// Run the backup
	err = RunBackup(cfg)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify that the old log file was backed up
	backupOldFilePath := filepath.Join(backupDir, "old.log")
	if _, err := os.Stat(backupOldFilePath); os.IsNotExist(err) {
		t.Errorf("Expected old log file to be backed up, but it was not found in backup directory")
	}

	// Verify that the old log file was pruned
	if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected old log file to be pruned (deleted), but it still exists")
	}

	// Verify that the new log file was not backed up or pruned
	backupNewFilePath := filepath.Join(backupDir, "new.log")
	if _, err := os.Stat(backupNewFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected new log file to not be backed up, but it was found in backup directory")
	}

	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Errorf("Expected new log file to not be pruned (deleted), but it is missing")
	}
}

// TestRunBackupPreservesDirectoryStructure tests that backup preserves directory structure
func TestRunBackupPreservesDirectoryStructure(t *testing.T) {
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp log directory: %v", err)
	}
	defer os.RemoveAll(logDir)

	backupDir, err := os.MkdirTemp("", "backupdir")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory: %v", err)
	}
	defer os.RemoveAll(backupDir)

	// Create subdirectories with files that have the same name
	subDir1 := filepath.Join(logDir, "app1", "logs")
	subDir2 := filepath.Join(logDir, "app2", "logs")
	if err := os.MkdirAll(subDir1, 0755); err != nil {
		t.Fatalf("Failed to create subdir1: %v", err)
	}
	if err := os.MkdirAll(subDir2, 0755); err != nil {
		t.Fatalf("Failed to create subdir2: %v", err)
	}

	// Create files with same name but different content in different directories
	file1 := filepath.Join(subDir1, "error.log")
	file2 := filepath.Join(subDir2, "error.log")
	content1 := []byte("error log from app1")
	content2 := []byte("error log from app2")

	if err := os.WriteFile(file1, content1, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, content2, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Set old modification time so files get backed up
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(file1, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set mod time on file1: %v", err)
	}
	if err := os.Chtimes(file2, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set mod time on file2: %v", err)
	}

	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPath:      backupDir,
		EnableBackup:    true,
		TargetFolder:    logDir,
		RemoteBackup:    "",
	}

	err = RunBackup(cfg)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify both files were backed up with directory structure preserved
	backupFile1 := filepath.Join(backupDir, "app1", "logs", "error.log")
	backupFile2 := filepath.Join(backupDir, "app2", "logs", "error.log")

	// Check file1 backup exists and has correct content
	backupContent1, err := os.ReadFile(backupFile1)
	if err != nil {
		t.Errorf("Expected backup file1 at %s, but got error: %v", backupFile1, err)
	} else if string(backupContent1) != string(content1) {
		t.Errorf("Backup file1 content mismatch: got %q, want %q", string(backupContent1), string(content1))
	}

	// Check file2 backup exists and has correct content
	backupContent2, err := os.ReadFile(backupFile2)
	if err != nil {
		t.Errorf("Expected backup file2 at %s, but got error: %v", backupFile2, err)
	} else if string(backupContent2) != string(content2) {
		t.Errorf("Backup file2 content mismatch: got %q, want %q", string(backupContent2), string(content2))
	}
}

// TestRunBackupNoBackupFlag tests the RunBackup function when backups are disabled
func TestRunBackupNoBackupFlag(t *testing.T) {
	// Create a temporary log directory
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp log directory: %v", err)
	}
	defer os.RemoveAll(logDir)

	// Create a test log file that should be pruned without backup
	filePath := filepath.Join(logDir, "test.log")
	if err := os.WriteFile(filePath, []byte("test log data"), 0644); err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}
	// Modify the file's modification time to be older than the prune threshold
	modTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatalf("Failed to set modification time on test log file: %v", err)
	}

	// Define the config with backups disabled
	cfg := &config.Config{
		PruneAfterHours: 24,    // Files older than 24 hours should be pruned
		BackupPath:      "",    // No backup directory since backup is disabled
		EnableBackup:    false, // Backup is disabled
		TargetFolder:    logDir,
	}

	// Run the backup with backup disabled
	err = RunBackup(cfg)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify that the log file was pruned without backup
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected log file to be pruned (deleted), but it still exists")
	}
}
