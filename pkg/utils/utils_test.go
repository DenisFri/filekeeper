package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteRemoteCopy_SourceNotExist(t *testing.T) {
	err := ExecuteRemoteCopy("/nonexistent/file.txt", "user@host:/path")
	if err == nil {
		t.Error("expected error for non-existent source file")
	}
}

func TestExecuteRemoteCopy_EmptyDestination(t *testing.T) {
	// Create a temp file
	tmpDir, err := os.MkdirTemp("", "utils_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = ExecuteRemoteCopy(tmpFile, "")
	if err == nil {
		t.Error("expected error for empty destination")
	}
}

func TestExecuteRemoteCopy_CommandInjectionPrevention(t *testing.T) {
	// Create a temp file
	tmpDir, err := os.MkdirTemp("", "utils_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// This malicious destination should NOT execute the injected command
	// With the old implementation using sh -c, this would execute "rm -rf /"
	// With the new implementation, it will be passed as a literal argument to scp
	maliciousDestination := "user@host:/path; rm -rf /"

	// The scp command will fail because it can't connect to the host,
	// but importantly, the injected command should NOT be executed
	err = ExecuteRemoteCopy(tmpFile, maliciousDestination)

	// We expect an error (scp will fail to connect), but no command injection
	if err == nil {
		t.Error("expected error from scp (connection failure), got nil")
	}

	// Verify no files were deleted (the injected rm command didn't run)
	if _, statErr := os.Stat(tmpFile); os.IsNotExist(statErr) {
		t.Error("temp file was deleted - possible command injection!")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "utils_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	destFile := filepath.Join(tmpDir, "dest.txt")
	if err := CopyFile(srcFile, destFile); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}

	if string(destContent) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(destContent), string(content))
	}
}
