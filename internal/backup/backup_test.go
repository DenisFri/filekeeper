package backup

import (
	"filekeeper/internal/config"
	"io/ioutil"
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

	backupDir, err := ioutil.TempDir("", "backupdir")
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
	if err := ioutil.WriteFile(newFilePath, []byte("new log data"), 0644); err != nil {
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

// TestRunBackupNoBackupFlag tests the RunBackup function when backups are disabled
func TestRunBackupNoBackupFlag(t *testing.T) {
	// Create a temporary log directory
	logDir, err := ioutil.TempDir("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp log directory: %v", err)
	}
	defer os.RemoveAll(logDir)

	// Create a test log file that should be pruned without backup
	filePath := filepath.Join(logDir, "test.log")
	if err := ioutil.WriteFile(filePath, []byte("test log data"), 0644); err != nil {
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
