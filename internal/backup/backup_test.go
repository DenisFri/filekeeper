package backup

import (
	"context"
	"filekeeper/internal/config"
	"filekeeper/internal/logger"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testLogger creates a logger for testing
func testLogger() *slog.Logger {
	return logger.New("info", "text")
}

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
	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify result
	if result.BackedUp != 1 {
		t.Errorf("Expected 1 file backed up, got %d", result.BackedUp)
	}
	if result.Pruned != 1 {
		t.Errorf("Expected 1 file pruned, got %d", result.Pruned)
	}
	if result.HasErrors() {
		t.Errorf("Expected no errors, got %d", result.Failed)
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

	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify result
	if result.BackedUp != 2 {
		t.Errorf("Expected 2 files backed up, got %d", result.BackedUp)
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
	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify result
	if result.Pruned != 1 {
		t.Errorf("Expected 1 file pruned, got %d", result.Pruned)
	}
	if result.BackedUp != 0 {
		t.Errorf("Expected 0 files backed up, got %d", result.BackedUp)
	}

	// Verify that the log file was pruned without backup
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected log file to be pruned (deleted), but it still exists")
	}
}

// TestRunBackupContextCancellation tests that backup respects context cancellation
func TestRunBackupContextCancellation(t *testing.T) {
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

	// Create a test log file
	filePath := filepath.Join(logDir, "test.log")
	if err := os.WriteFile(filePath, []byte("test log data"), 0644); err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filePath, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set modification time: %v", err)
	}

	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPath:      backupDir,
		EnableBackup:    true,
		TargetFolder:    logDir,
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	log := testLogger()
	_, err = RunBackup(ctx, cfg, nil, log)

	// Should return context.Canceled error
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestRunBackupReturnsResult tests that RunBackup returns a valid result
func TestRunBackupReturnsResult(t *testing.T) {
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

	// Create multiple test files
	for i := 0; i < 3; i++ {
		filePath := filepath.Join(logDir, filepath.Base(logDir)+string(rune('a'+i))+".log")
		if err := os.WriteFile(filePath, []byte("test log data"), 0644); err != nil {
			t.Fatalf("Failed to create test log file: %v", err)
		}
		oldModTime := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(filePath, oldModTime, oldModTime); err != nil {
			t.Fatalf("Failed to set modification time: %v", err)
		}
	}

	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPath:      backupDir,
		EnableBackup:    true,
		TargetFolder:    logDir,
	}

	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify result contains expected counts
	if result.BackedUp != 3 {
		t.Errorf("Expected 3 files backed up, got %d", result.BackedUp)
	}
	if result.Pruned != 3 {
		t.Errorf("Expected 3 files pruned, got %d", result.Pruned)
	}
	if result.HasErrors() {
		t.Errorf("Expected no errors, got %d", result.Failed)
	}
	if result.TotalBytes == 0 {
		t.Error("Expected TotalBytes to be > 0")
	}
}

// TestRunBackupDryRun tests that dry-run mode doesn't modify files
func TestRunBackupDryRun(t *testing.T) {
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

	// Create a test log file that would be backed up and pruned
	oldFilePath := filepath.Join(logDir, "old.log")
	if err := os.WriteFile(oldFilePath, []byte("old log data"), 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFilePath, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set modification time: %v", err)
	}

	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPath:      backupDir,
		EnableBackup:    true,
		TargetFolder:    logDir,
	}

	// Run in dry-run mode
	ctx := context.Background()
	log := testLogger()
	opts := &RunOptions{DryRun: true}
	result, err := RunBackup(ctx, cfg, opts, log)
	if err != nil {
		t.Fatalf("RunBackup dry-run failed: %v", err)
	}

	// Result should still report what would be done
	if result.Pruned != 1 {
		t.Errorf("Expected 1 file would be pruned, got %d", result.Pruned)
	}

	// But the original file should still exist (not pruned)
	if _, err := os.Stat(oldFilePath); os.IsNotExist(err) {
		t.Errorf("Original file was deleted in dry-run mode!")
	}

	// And no backup should have been created
	backupOldFilePath := filepath.Join(backupDir, "old.log")
	if _, err := os.Stat(backupOldFilePath); !os.IsNotExist(err) {
		t.Errorf("Backup file was created in dry-run mode!")
	}
}

// TestRunBackupMultipleLocalDestinations tests backup to multiple local destinations
func TestRunBackupMultipleLocalDestinations(t *testing.T) {
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp log directory: %v", err)
	}
	defer os.RemoveAll(logDir)

	// Create two backup directories
	backupDir1, err := os.MkdirTemp("", "backupdir1")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory 1: %v", err)
	}
	defer os.RemoveAll(backupDir1)

	backupDir2, err := os.MkdirTemp("", "backupdir2")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory 2: %v", err)
	}
	defer os.RemoveAll(backupDir2)

	// Create a test log file
	oldFilePath := filepath.Join(logDir, "old.log")
	content := []byte("old log data for multi-dest test")
	if err := os.WriteFile(oldFilePath, content, 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFilePath, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set modification time: %v", err)
	}

	// Config with multiple backup paths
	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPaths:     []string{backupDir1, backupDir2},
		EnableBackup:    true,
		TargetFolder:    logDir,
	}

	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify result
	if result.BackedUp != 1 {
		t.Errorf("Expected 1 file backed up, got %d", result.BackedUp)
	}

	// Verify file was backed up to both destinations
	backupFile1 := filepath.Join(backupDir1, "old.log")
	backupFile2 := filepath.Join(backupDir2, "old.log")

	content1, err := os.ReadFile(backupFile1)
	if err != nil {
		t.Errorf("Expected backup in dir1, got error: %v", err)
	} else if string(content1) != string(content) {
		t.Errorf("Backup1 content mismatch")
	}

	content2, err := os.ReadFile(backupFile2)
	if err != nil {
		t.Errorf("Expected backup in dir2, got error: %v", err)
	} else if string(content2) != string(content) {
		t.Errorf("Backup2 content mismatch")
	}

	// Verify original was pruned
	if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
		t.Errorf("Original file should be pruned")
	}
}

// TestRunBackupMixedPathConfig tests backward compatibility with both single and array paths
func TestRunBackupMixedPathConfig(t *testing.T) {
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp log directory: %v", err)
	}
	defer os.RemoveAll(logDir)

	// Create three backup directories
	backupDir1, err := os.MkdirTemp("", "backupdir1")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory 1: %v", err)
	}
	defer os.RemoveAll(backupDir1)

	backupDir2, err := os.MkdirTemp("", "backupdir2")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory 2: %v", err)
	}
	defer os.RemoveAll(backupDir2)

	backupDir3, err := os.MkdirTemp("", "backupdir3")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory 3: %v", err)
	}
	defer os.RemoveAll(backupDir3)

	// Create a test log file
	oldFilePath := filepath.Join(logDir, "old.log")
	content := []byte("test content")
	if err := os.WriteFile(oldFilePath, content, 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFilePath, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set modification time: %v", err)
	}

	// Config with both single path and array paths (backward compatible)
	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPath:      backupDir1,                       // Single path (backward compatible)
		BackupPaths:     []string{backupDir2, backupDir3}, // Array paths
		EnableBackup:    true,
		TargetFolder:    logDir,
	}

	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	if result.BackedUp != 1 {
		t.Errorf("Expected 1 file backed up, got %d", result.BackedUp)
	}

	// Verify file was backed up to all three destinations
	for i, backupDir := range []string{backupDir1, backupDir2, backupDir3} {
		backupFile := filepath.Join(backupDir, "old.log")
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			t.Errorf("Expected backup in dir%d (%s), but file not found", i+1, backupDir)
		}
	}
}

// TestRunBackupPartialLocalFailure tests that backup continues if one local destination fails
func TestRunBackupPartialLocalFailure(t *testing.T) {
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp log directory: %v", err)
	}
	defer os.RemoveAll(logDir)

	// Create one valid backup directory
	backupDir1, err := os.MkdirTemp("", "backupdir1")
	if err != nil {
		t.Fatalf("Failed to create temp backup directory 1: %v", err)
	}
	defer os.RemoveAll(backupDir1)

	// Use a non-existent path that will fail (but not too long to avoid path issues)
	invalidBackupDir := filepath.Join(os.TempDir(), "nonexistent_parent_12345", "child")

	// Create a test log file
	oldFilePath := filepath.Join(logDir, "old.log")
	content := []byte("test content")
	if err := os.WriteFile(oldFilePath, content, 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFilePath, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set modification time: %v", err)
	}

	// Config with one valid and one invalid backup path
	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPaths:     []string{backupDir1, invalidBackupDir},
		EnableBackup:    true,
		TargetFolder:    logDir,
	}

	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)

	// Should succeed because at least one destination worked
	if err != nil {
		t.Fatalf("RunBackup should succeed with partial failure, got: %v", err)
	}

	if result.BackedUp != 1 {
		t.Errorf("Expected 1 file backed up, got %d", result.BackedUp)
	}

	// Verify file was backed up to the valid destination
	backupFile := filepath.Join(backupDir1, "old.log")
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Errorf("Expected backup in valid dir, but file not found")
	}
}

// TestRunBackupWithCompression tests backup with gzip compression enabled
func TestRunBackupWithCompression(t *testing.T) {
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

	// Create a test log file with compressible content
	oldFilePath := filepath.Join(logDir, "old.log")
	// Repeated content compresses well
	content := []byte("This is test content that will compress well. " +
		"Repeated patterns help compression. " +
		"This is test content that will compress well. " +
		"Repeated patterns help compression. ")
	for i := 0; i < 100; i++ {
		content = append(content, content...)
		if len(content) > 10000 {
			break
		}
	}

	if err := os.WriteFile(oldFilePath, content, 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFilePath, oldModTime, oldModTime); err != nil {
		t.Fatalf("Failed to set modification time: %v", err)
	}

	// Config with compression enabled
	cfg := &config.Config{
		PruneAfterHours: 24,
		BackupPath:      backupDir,
		EnableBackup:    true,
		TargetFolder:    logDir,
		Compression: &config.CompressionConfig{
			Enabled:   true,
			Algorithm: "gzip",
			Level:     6,
		},
	}

	ctx := context.Background()
	log := testLogger()
	result, err := RunBackup(ctx, cfg, nil, log)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify result
	if result.BackedUp != 1 {
		t.Errorf("Expected 1 file backed up, got %d", result.BackedUp)
	}

	// Verify compressed file exists with .gz extension
	compressedFile := filepath.Join(backupDir, "old.log.gz")
	if _, err := os.Stat(compressedFile); os.IsNotExist(err) {
		t.Errorf("Expected compressed file at %s, but not found", compressedFile)
	}

	// Verify uncompressed file does NOT exist
	uncompressedFile := filepath.Join(backupDir, "old.log")
	if _, err := os.Stat(uncompressedFile); !os.IsNotExist(err) {
		t.Errorf("Uncompressed file should not exist at %s", uncompressedFile)
	}

	// Verify compression statistics
	if result.OriginalBytes == 0 {
		t.Error("Expected OriginalBytes > 0")
	}
	if result.CompressedBytes == 0 {
		t.Error("Expected CompressedBytes > 0")
	}
	if result.CompressedBytes >= result.OriginalBytes {
		t.Errorf("Expected compression to reduce size: original=%d, compressed=%d",
			result.OriginalBytes, result.CompressedBytes)
	}

	t.Logf("Compression: original=%d, compressed=%d, ratio=%.1f%%, saved=%.1f%%",
		result.OriginalBytes, result.CompressedBytes,
		result.CompressionRatio(), result.SpaceSaved())
}

func TestRunBackupArchiveMode(t *testing.T) {
	// Create temp directories
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(logDir)

	backupDir, err := os.MkdirTemp("", "backupdir")
	if err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}
	defer os.RemoveAll(backupDir)

	// Create multiple old test files
	oldTime := time.Now().Add(-48 * time.Hour)
	files := []string{"file1.log", "file2.log", "file3.log"}
	for _, name := range files {
		filePath := filepath.Join(logDir, name)
		content := strings.Repeat(fmt.Sprintf("Content of %s\n", name), 100)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := os.Chtimes(filePath, oldTime, oldTime); err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}
	}

	// Config with archive mode enabled
	cfg := &config.Config{
		PruneAfterHours: 24,
		TargetFolder:    logDir,
		RunInterval:     60,
		BackupPath:      backupDir,
		EnableBackup:    true,
		Archive: &config.ArchiveConfig{
			Enabled: true,
			Format:  "tar.gz",
			GroupBy: "daily",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	result, err := RunBackup(context.Background(), cfg, nil, logger)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify all files were backed up
	if result.BackedUp != 3 {
		t.Errorf("Expected 3 files backed up, got %d", result.BackedUp)
	}

	// Verify archive exists
	archiveFiles, err := filepath.Glob(filepath.Join(backupDir, "backup-*.tar.gz"))
	if err != nil {
		t.Fatalf("Failed to glob archive files: %v", err)
	}
	if len(archiveFiles) != 1 {
		t.Errorf("Expected 1 archive file, got %d", len(archiveFiles))
	}

	// Verify archive statistics
	if result.ArchiveSize == 0 {
		t.Error("Expected ArchiveSize > 0")
	}
	if result.OriginalBytes == 0 {
		t.Error("Expected OriginalBytes > 0")
	}

	t.Logf("Archive: files=%d, original=%d, archive_size=%d, ratio=%.1f%%",
		result.BackedUp, result.OriginalBytes, result.ArchiveSize, result.CompressionRatio())
}

func TestRunBackupArchiveModeDryRun(t *testing.T) {
	// Create temp directories
	logDir, err := os.MkdirTemp("", "logdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(logDir)

	backupDir, err := os.MkdirTemp("", "backupdir")
	if err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}
	defer os.RemoveAll(backupDir)

	// Create an old test file
	oldFile := filepath.Join(logDir, "old.log")
	content := strings.Repeat("Old content\n", 100)
	if err := os.WriteFile(oldFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create old file: %v", err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set file time: %v", err)
	}

	// Config with archive mode enabled
	cfg := &config.Config{
		PruneAfterHours: 24,
		TargetFolder:    logDir,
		RunInterval:     60,
		BackupPath:      backupDir,
		EnableBackup:    true,
		Archive: &config.ArchiveConfig{
			Enabled: true,
			Format:  "zip",
			GroupBy: "weekly",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	opts := &RunOptions{DryRun: true}
	result, err := RunBackup(context.Background(), cfg, opts, logger)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// In dry-run mode, no files should be backed up
	if result.BackedUp != 0 {
		t.Errorf("Expected 0 files backed up in dry-run, got %d", result.BackedUp)
	}

	// Verify no archive was created
	archiveFiles, err := filepath.Glob(filepath.Join(backupDir, "backup-*"))
	if err != nil {
		t.Fatalf("Failed to glob archive files: %v", err)
	}
	if len(archiveFiles) != 0 {
		t.Errorf("Expected no archive files in dry-run, got %d", len(archiveFiles))
	}

	// Verify source file still exists (not pruned in dry-run)
	if _, err := os.Stat(oldFile); os.IsNotExist(err) {
		t.Error("Source file should still exist in dry-run mode")
	}
}
