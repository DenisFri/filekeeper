package compression

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Algorithm represents a compression algorithm type.
type Algorithm string

const (
	None Algorithm = "none"
	Gzip Algorithm = "gzip"
)

// Config holds compression configuration.
type Config struct {
	Enabled   bool      `json:"enabled"`
	Algorithm Algorithm `json:"algorithm"`
	Level     int       `json:"level"` // gzip: 1 (fastest) to 9 (best), default 6
}

// DefaultConfig returns the default compression configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:   false,
		Algorithm: None,
		Level:     gzip.DefaultCompression, // 6
	}
}

// Validate checks that the compression configuration is valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	switch c.Algorithm {
	case None, "":
		// No compression, nothing to validate
	case Gzip:
		if c.Level < gzip.HuffmanOnly || c.Level > gzip.BestCompression {
			// gzip.HuffmanOnly = -2, gzip.BestCompression = 9
			// Allow DefaultCompression (-1) and 1-9
			if c.Level != gzip.DefaultCompression && (c.Level < 1 || c.Level > 9) {
				return fmt.Errorf("gzip compression level must be between 1 and 9, got %d", c.Level)
			}
		}
	default:
		return fmt.Errorf("unknown compression algorithm: %s (supported: none, gzip)", c.Algorithm)
	}

	return nil
}

// Result contains compression statistics for a single file.
type Result struct {
	OriginalSize   int64
	CompressedSize int64
	Algorithm      Algorithm
}

// CompressionRatio returns the compression ratio as a percentage.
// A ratio of 80% means the compressed file is 80% of the original size.
func (r *Result) CompressionRatio() float64 {
	if r.OriginalSize == 0 {
		return 100
	}
	return float64(r.CompressedSize) / float64(r.OriginalSize) * 100
}

// SpaceSaved returns the percentage of space saved.
// A value of 20% means 20% of the original size was saved.
func (r *Result) SpaceSaved() float64 {
	return 100 - r.CompressionRatio()
}

// ExtensionFor returns the file extension for the given algorithm.
func ExtensionFor(alg Algorithm) string {
	switch alg {
	case Gzip:
		return ".gz"
	default:
		return ""
	}
}

// CompressFile compresses a source file to the destination using the configured algorithm.
// Returns compression statistics and any error encountered.
// If compression is disabled or algorithm is "none", performs a regular file copy.
func CompressFile(src, dest string, cfg *Config) (*Result, error) {
	// Get source file info for original size
	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil, fmt.Errorf("stat source file: %w", err)
	}

	result := &Result{
		OriginalSize: srcInfo.Size(),
		Algorithm:    None,
	}

	// If compression is disabled or algorithm is none, do regular copy
	if cfg == nil || !cfg.Enabled || cfg.Algorithm == None || cfg.Algorithm == "" {
		if err := copyFile(src, dest); err != nil {
			return nil, err
		}
		result.CompressedSize = srcInfo.Size()
		return result, nil
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return nil, fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()

	// Add appropriate extension to destination
	destPath := dest + ExtensionFor(cfg.Algorithm)

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("create destination file: %w", err)
	}
	defer destFile.Close()

	// Compress based on algorithm
	switch cfg.Algorithm {
	case Gzip:
		level := cfg.Level
		if level == 0 {
			level = gzip.DefaultCompression
		}
		writer, err := gzip.NewWriterLevel(destFile, level)
		if err != nil {
			return nil, fmt.Errorf("create gzip writer: %w", err)
		}

		if _, err := io.Copy(writer, srcFile); err != nil {
			writer.Close()
			return nil, fmt.Errorf("compress file: %w", err)
		}

		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close gzip writer: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown compression algorithm: %s", cfg.Algorithm)
	}

	// Get compressed file size
	destInfo, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("stat compressed file: %w", err)
	}

	result.Algorithm = cfg.Algorithm
	result.CompressedSize = destInfo.Size()

	return result, nil
}

// GetDestinationPath returns the destination path with compression extension if applicable.
func GetDestinationPath(dest string, cfg *Config) string {
	if cfg == nil || !cfg.Enabled || cfg.Algorithm == None || cfg.Algorithm == "" {
		return dest
	}
	return dest + ExtensionFor(cfg.Algorithm)
}

// DecompressFile decompresses a file to the destination.
// It auto-detects the algorithm from the file extension.
func DecompressFile(src, dest string) error {
	// Detect algorithm from extension
	ext := strings.ToLower(filepath.Ext(src))

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer destFile.Close()

	switch ext {
	case ".gz":
		reader, err := gzip.NewReader(srcFile)
		if err != nil {
			return fmt.Errorf("create gzip reader: %w", err)
		}
		defer reader.Close()

		if _, err := io.Copy(destFile, reader); err != nil {
			return fmt.Errorf("decompress file: %w", err)
		}
	default:
		// No compression, just copy
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
	}

	return nil
}

// copyFile performs a simple file copy without compression.
func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return nil
}
