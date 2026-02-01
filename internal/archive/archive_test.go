package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateArchiveName(t *testing.T) {
	testTime := time.Date(2026, 1, 24, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		groupBy  GroupBy
		format   Format
		expected string
	}{
		{
			name:     "daily tar.gz",
			groupBy:  GroupByDaily,
			format:   FormatTarGz,
			expected: "backup-2026-01-24.tar.gz",
		},
		{
			name:     "weekly tar.gz",
			groupBy:  GroupByWeekly,
			format:   FormatTarGz,
			expected: "backup-2026-W04.tar.gz",
		},
		{
			name:     "monthly tar.gz",
			groupBy:  GroupByMonthly,
			format:   FormatTarGz,
			expected: "backup-2026-01.tar.gz",
		},
		{
			name:     "daily tar",
			groupBy:  GroupByDaily,
			format:   FormatTar,
			expected: "backup-2026-01-24.tar",
		},
		{
			name:     "daily zip",
			groupBy:  GroupByDaily,
			format:   FormatZip,
			expected: "backup-2026-01-24.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateArchiveName(testTime, tt.groupBy, tt.format)
			if got != tt.expected {
				t.Errorf("GenerateArchiveName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "disabled",
			config:  &Config{Enabled: false},
			wantErr: false,
		},
		{
			name:    "valid tar.gz",
			config:  &Config{Enabled: true, Format: FormatTarGz, GroupBy: GroupByDaily},
			wantErr: false,
		},
		{
			name:    "valid tar",
			config:  &Config{Enabled: true, Format: FormatTar, GroupBy: GroupByWeekly},
			wantErr: false,
		},
		{
			name:    "valid zip",
			config:  &Config{Enabled: true, Format: FormatZip, GroupBy: GroupByMonthly},
			wantErr: false,
		},
		{
			name:    "invalid format",
			config:  &Config{Enabled: true, Format: "rar"},
			wantErr: true,
		},
		{
			name:    "invalid group_by",
			config:  &Config{Enabled: true, Format: FormatTarGz, GroupBy: "yearly"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateTarGzArchive(t *testing.T) {
	// Create temp directories
	srcDir, err := os.MkdirTemp("", "archive_src")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	outDir, err := os.MkdirTemp("", "archive_out")
	if err != nil {
		t.Fatalf("Failed to create temp out dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	// Create test files
	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(srcDir, "subdir", "file2.txt")
	content1 := strings.Repeat("Content of file 1. ", 100)
	content2 := strings.Repeat("Content of file 2. ", 100)

	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(file2), 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Create archive
	cfg := &Config{
		Enabled: true,
		Format:  FormatTarGz,
		GroupBy: GroupByDaily,
	}
	creator := NewCreator(cfg, outDir)

	files := map[string]string{
		file1: "file1.txt",
		file2: "subdir/file2.txt",
	}

	archiveTime := time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC)
	result, err := creator.CreateArchive(files, archiveTime)
	if err != nil {
		t.Fatalf("CreateArchive failed: %v", err)
	}

	// Verify result
	if result.FilesArchived != 2 {
		t.Errorf("Expected 2 files archived, got %d", result.FilesArchived)
	}

	expectedPath := filepath.Join(outDir, "backup-2026-01-24.tar.gz")
	if result.ArchivePath != expectedPath {
		t.Errorf("Expected archive path %s, got %s", expectedPath, result.ArchivePath)
	}

	if result.ArchiveSize == 0 {
		t.Error("Expected archive size > 0")
	}

	// Archive should be smaller than original (compressed)
	if result.ArchiveSize >= result.TotalSize {
		t.Logf("Note: Archive size (%d) not smaller than original (%d)", result.ArchiveSize, result.TotalSize)
	}

	// Verify archive exists
	if _, err := os.Stat(result.ArchivePath); os.IsNotExist(err) {
		t.Errorf("Archive file not found at %s", result.ArchivePath)
	}

	// Extract and verify
	extractDir, err := os.MkdirTemp("", "archive_extract")
	if err != nil {
		t.Fatalf("Failed to create extract dir: %v", err)
	}
	defer os.RemoveAll(extractDir)

	if err := ExtractArchive(result.ArchivePath, extractDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	// Verify extracted files
	extracted1 := filepath.Join(extractDir, "file1.txt")
	extracted2 := filepath.Join(extractDir, "subdir", "file2.txt")

	content, err := os.ReadFile(extracted1)
	if err != nil {
		t.Errorf("Failed to read extracted file1: %v", err)
	} else if string(content) != content1 {
		t.Errorf("Extracted file1 content mismatch")
	}

	content, err = os.ReadFile(extracted2)
	if err != nil {
		t.Errorf("Failed to read extracted file2: %v", err)
	} else if string(content) != content2 {
		t.Errorf("Extracted file2 content mismatch")
	}
}

func TestCreateZipArchive(t *testing.T) {
	// Create temp directories
	srcDir, err := os.MkdirTemp("", "archive_src")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	outDir, err := os.MkdirTemp("", "archive_out")
	if err != nil {
		t.Fatalf("Failed to create temp out dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	// Create test file
	file1 := filepath.Join(srcDir, "test.txt")
	content := strings.Repeat("Test content for zip. ", 100)
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create zip archive
	cfg := &Config{
		Enabled: true,
		Format:  FormatZip,
		GroupBy: GroupByMonthly,
	}
	creator := NewCreator(cfg, outDir)

	files := map[string]string{
		file1: "test.txt",
	}

	archiveTime := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	result, err := creator.CreateArchive(files, archiveTime)
	if err != nil {
		t.Fatalf("CreateArchive failed: %v", err)
	}

	// Verify result
	if result.FilesArchived != 1 {
		t.Errorf("Expected 1 file archived, got %d", result.FilesArchived)
	}

	expectedPath := filepath.Join(outDir, "backup-2026-02.zip")
	if result.ArchivePath != expectedPath {
		t.Errorf("Expected archive path %s, got %s", expectedPath, result.ArchivePath)
	}

	// Extract and verify
	extractDir, err := os.MkdirTemp("", "archive_extract")
	if err != nil {
		t.Fatalf("Failed to create extract dir: %v", err)
	}
	defer os.RemoveAll(extractDir)

	if err := ExtractArchive(result.ArchivePath, extractDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	extracted := filepath.Join(extractDir, "test.txt")
	extractedContent, err := os.ReadFile(extracted)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
	} else if string(extractedContent) != content {
		t.Errorf("Extracted content mismatch")
	}
}

func TestCreateTarArchive(t *testing.T) {
	// Create temp directories
	srcDir, err := os.MkdirTemp("", "archive_src")
	if err != nil {
		t.Fatalf("Failed to create temp src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	outDir, err := os.MkdirTemp("", "archive_out")
	if err != nil {
		t.Fatalf("Failed to create temp out dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	// Create test file
	file1 := filepath.Join(srcDir, "test.txt")
	content := "Test content for tar"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create tar archive (no compression)
	cfg := &Config{
		Enabled: true,
		Format:  FormatTar,
		GroupBy: GroupByWeekly,
	}
	creator := NewCreator(cfg, outDir)

	files := map[string]string{
		file1: "test.txt",
	}

	archiveTime := time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC)
	result, err := creator.CreateArchive(files, archiveTime)
	if err != nil {
		t.Fatalf("CreateArchive failed: %v", err)
	}

	expectedPath := filepath.Join(outDir, "backup-2026-W04.tar")
	if result.ArchivePath != expectedPath {
		t.Errorf("Expected archive path %s, got %s", expectedPath, result.ArchivePath)
	}

	// Extract and verify
	extractDir, err := os.MkdirTemp("", "archive_extract")
	if err != nil {
		t.Fatalf("Failed to create extract dir: %v", err)
	}
	defer os.RemoveAll(extractDir)

	if err := ExtractArchive(result.ArchivePath, extractDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	extracted := filepath.Join(extractDir, "test.txt")
	extractedContent, err := os.ReadFile(extracted)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
	} else if string(extractedContent) != content {
		t.Errorf("Extracted content mismatch")
	}
}

func TestCreateArchiveEmptyFiles(t *testing.T) {
	outDir, err := os.MkdirTemp("", "archive_out")
	if err != nil {
		t.Fatalf("Failed to create temp out dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	cfg := &Config{
		Enabled: true,
		Format:  FormatTarGz,
		GroupBy: GroupByDaily,
	}
	creator := NewCreator(cfg, outDir)

	// Empty files map
	files := map[string]string{}

	result, err := creator.CreateArchive(files, time.Now())
	if err != nil {
		t.Fatalf("CreateArchive with empty files failed: %v", err)
	}

	if result.FilesArchived != 0 {
		t.Errorf("Expected 0 files archived, got %d", result.FilesArchived)
	}
}

func TestExtensionFor(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
	}{
		{FormatTar, ".tar"},
		{FormatTarGz, ".tar.gz"},
		{FormatZip, ".zip"},
		{"unknown", ".tar.gz"}, // default
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got := ExtensionFor(tt.format)
			if got != tt.expected {
				t.Errorf("ExtensionFor(%s) = %q, want %q", tt.format, got, tt.expected)
			}
		})
	}
}

func TestCompressionRatio(t *testing.T) {
	result := &Result{
		TotalSize:   1000,
		ArchiveSize: 200,
	}

	ratio := result.CompressionRatio()
	if ratio != 20.0 {
		t.Errorf("CompressionRatio() = %.1f, want 20.0", ratio)
	}

	// Test zero original size
	result2 := &Result{
		TotalSize:   0,
		ArchiveSize: 0,
	}
	if result2.CompressionRatio() != 100.0 {
		t.Errorf("CompressionRatio() with zero = %.1f, want 100.0", result2.CompressionRatio())
	}
}
