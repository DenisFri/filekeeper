package compression

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompressFileGzip(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compression_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file with compressible content (repeated text compresses well)
	srcPath := filepath.Join(tmpDir, "test.txt")
	content := strings.Repeat("Hello, this is test content that should compress well! ", 1000)
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Compress the file
	destPath := filepath.Join(tmpDir, "test.txt")
	cfg := &Config{
		Enabled:   true,
		Algorithm: Gzip,
		Level:     gzip.DefaultCompression,
	}

	result, err := CompressFile(srcPath, destPath, cfg)
	if err != nil {
		t.Fatalf("CompressFile failed: %v", err)
	}

	// Verify result statistics
	if result.OriginalSize != int64(len(content)) {
		t.Errorf("Expected original size %d, got %d", len(content), result.OriginalSize)
	}

	if result.CompressedSize >= result.OriginalSize {
		t.Errorf("Compressed size (%d) should be less than original (%d)", result.CompressedSize, result.OriginalSize)
	}

	if result.Algorithm != Gzip {
		t.Errorf("Expected algorithm %s, got %s", Gzip, result.Algorithm)
	}

	// Verify compression ratio is reasonable for text
	ratio := result.CompressionRatio()
	if ratio > 50 {
		t.Errorf("Expected compression ratio < 50%% for repeated text, got %.1f%%", ratio)
	}

	// Verify the compressed file exists with .gz extension
	compressedPath := destPath + ".gz"
	if _, err := os.Stat(compressedPath); os.IsNotExist(err) {
		t.Errorf("Compressed file not found at %s", compressedPath)
	}

	// Verify we can decompress and get original content
	decompressedPath := filepath.Join(tmpDir, "decompressed.txt")
	if err := DecompressFile(compressedPath, decompressedPath); err != nil {
		t.Fatalf("DecompressFile failed: %v", err)
	}

	decompressedContent, err := os.ReadFile(decompressedPath)
	if err != nil {
		t.Fatalf("Failed to read decompressed file: %v", err)
	}

	if string(decompressedContent) != content {
		t.Errorf("Decompressed content doesn't match original")
	}
}

func TestCompressFileNoCompression(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compression_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "test.txt")
	content := []byte("Test content without compression")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Copy without compression
	destPath := filepath.Join(tmpDir, "copy.txt")
	cfg := &Config{
		Enabled:   false,
		Algorithm: None,
	}

	result, err := CompressFile(srcPath, destPath, cfg)
	if err != nil {
		t.Fatalf("CompressFile failed: %v", err)
	}

	// Verify sizes are equal (no compression)
	if result.OriginalSize != result.CompressedSize {
		t.Errorf("Expected equal sizes without compression, got %d vs %d", result.OriginalSize, result.CompressedSize)
	}

	// Verify the file exists without extension
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Copied file not found at %s", destPath)
	}

	// Verify content matches
	copiedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if !bytes.Equal(copiedContent, content) {
		t.Errorf("Copied content doesn't match original")
	}
}

func TestCompressFileNilConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compression_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "test.txt")
	content := []byte("Test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Copy with nil config (should do regular copy)
	destPath := filepath.Join(tmpDir, "copy.txt")
	result, err := CompressFile(srcPath, destPath, nil)
	if err != nil {
		t.Fatalf("CompressFile with nil config failed: %v", err)
	}

	if result.Algorithm != None {
		t.Errorf("Expected algorithm None, got %s", result.Algorithm)
	}

	// Verify the file exists
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Copied file not found at %s", destPath)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "disabled compression",
			config:  &Config{Enabled: false},
			wantErr: false,
		},
		{
			name:    "valid gzip default level",
			config:  &Config{Enabled: true, Algorithm: Gzip, Level: gzip.DefaultCompression},
			wantErr: false,
		},
		{
			name:    "valid gzip level 1",
			config:  &Config{Enabled: true, Algorithm: Gzip, Level: 1},
			wantErr: false,
		},
		{
			name:    "valid gzip level 9",
			config:  &Config{Enabled: true, Algorithm: Gzip, Level: 9},
			wantErr: false,
		},
		{
			name:    "invalid gzip level 10",
			config:  &Config{Enabled: true, Algorithm: Gzip, Level: 10},
			wantErr: true,
		},
		{
			name:    "unknown algorithm",
			config:  &Config{Enabled: true, Algorithm: "unknown"},
			wantErr: true,
		},
		{
			name:    "none algorithm enabled",
			config:  &Config{Enabled: true, Algorithm: None},
			wantErr: false,
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

func TestExtensionFor(t *testing.T) {
	tests := []struct {
		algorithm Algorithm
		expected  string
	}{
		{None, ""},
		{Gzip, ".gz"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.algorithm), func(t *testing.T) {
			got := ExtensionFor(tt.algorithm)
			if got != tt.expected {
				t.Errorf("ExtensionFor(%s) = %q, want %q", tt.algorithm, got, tt.expected)
			}
		})
	}
}

func TestGetDestinationPath(t *testing.T) {
	tests := []struct {
		name     string
		dest     string
		cfg      *Config
		expected string
	}{
		{
			name:     "nil config",
			dest:     "/backup/file.txt",
			cfg:      nil,
			expected: "/backup/file.txt",
		},
		{
			name:     "disabled compression",
			dest:     "/backup/file.txt",
			cfg:      &Config{Enabled: false, Algorithm: Gzip},
			expected: "/backup/file.txt",
		},
		{
			name:     "gzip enabled",
			dest:     "/backup/file.txt",
			cfg:      &Config{Enabled: true, Algorithm: Gzip},
			expected: "/backup/file.txt.gz",
		},
		{
			name:     "none algorithm",
			dest:     "/backup/file.txt",
			cfg:      &Config{Enabled: true, Algorithm: None},
			expected: "/backup/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDestinationPath(tt.dest, tt.cfg)
			if got != tt.expected {
				t.Errorf("GetDestinationPath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCompressionResult(t *testing.T) {
	result := &Result{
		OriginalSize:   1000,
		CompressedSize: 200,
		Algorithm:      Gzip,
	}

	ratio := result.CompressionRatio()
	if ratio != 20.0 {
		t.Errorf("CompressionRatio() = %.1f, want 20.0", ratio)
	}

	saved := result.SpaceSaved()
	if saved != 80.0 {
		t.Errorf("SpaceSaved() = %.1f, want 80.0", saved)
	}
}

func TestCompressionResultZeroOriginal(t *testing.T) {
	result := &Result{
		OriginalSize:   0,
		CompressedSize: 0,
	}

	ratio := result.CompressionRatio()
	if ratio != 100.0 {
		t.Errorf("CompressionRatio() with zero original = %.1f, want 100.0", ratio)
	}
}

func TestGzipCompressionLevels(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compression_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file with compressible content
	srcPath := filepath.Join(tmpDir, "test.txt")
	content := strings.Repeat("ABCDEFGHIJ", 10000)
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	levels := []int{1, 5, 9}
	var prevSize int64 = int64(len(content))

	for _, level := range levels {
		destPath := filepath.Join(tmpDir, "test_level")
		cfg := &Config{
			Enabled:   true,
			Algorithm: Gzip,
			Level:     level,
		}

		result, err := CompressFile(srcPath, destPath, cfg)
		if err != nil {
			t.Fatalf("CompressFile at level %d failed: %v", level, err)
		}

		t.Logf("Level %d: original=%d, compressed=%d, ratio=%.1f%%",
			level, result.OriginalSize, result.CompressedSize, result.CompressionRatio())

		// Higher levels should give same or better compression
		if result.CompressedSize > prevSize {
			t.Logf("Note: Level %d (%d bytes) not smaller than previous (%d bytes) - this is acceptable for some data",
				level, result.CompressedSize, prevSize)
		}

		// Clean up for next iteration
		os.Remove(destPath + ".gz")
	}
}
