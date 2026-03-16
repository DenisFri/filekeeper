package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Format represents the archive format type.
type Format string

const (
	FormatTar   Format = "tar"
	FormatTarGz Format = "tar.gz"
	FormatZip   Format = "zip"
)

// GroupBy represents how files are grouped into archives.
type GroupBy string

const (
	GroupByDaily   GroupBy = "daily"
	GroupByWeekly  GroupBy = "weekly"
	GroupByMonthly GroupBy = "monthly"
)

// Config holds archive configuration.
type Config struct {
	Enabled bool    `json:"enabled"`
	Format  Format  `json:"format"`   // tar, tar.gz, zip
	GroupBy GroupBy `json:"group_by"` // daily, weekly, monthly
}

// DefaultConfig returns the default archive configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: false,
		Format:  FormatTarGz,
		GroupBy: GroupByDaily,
	}
}

// Validate checks that the archive configuration is valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	switch c.Format {
	case FormatTar, FormatTarGz, FormatZip, "":
		// Valid formats
	default:
		return fmt.Errorf("unknown archive format: %s (supported: tar, tar.gz, zip)", c.Format)
	}

	switch c.GroupBy {
	case GroupByDaily, GroupByWeekly, GroupByMonthly, "":
		// Valid groupings
	default:
		return fmt.Errorf("unknown group_by value: %s (supported: daily, weekly, monthly)", c.GroupBy)
	}

	return nil
}

// ExtensionFor returns the file extension for the given format.
func ExtensionFor(format Format) string {
	switch format {
	case FormatTar:
		return ".tar"
	case FormatTarGz:
		return ".tar.gz"
	case FormatZip:
		return ".zip"
	default:
		return ".tar.gz"
	}
}

// GenerateArchiveName generates an archive name based on the grouping and timestamp.
func GenerateArchiveName(t time.Time, groupBy GroupBy, format Format) string {
	var datePart string

	switch groupBy {
	case GroupByDaily:
		datePart = t.Format("2006-01-02")
	case GroupByWeekly:
		year, week := t.ISOWeek()
		datePart = fmt.Sprintf("%d-W%02d", year, week)
	case GroupByMonthly:
		datePart = t.Format("2006-01")
	default:
		datePart = t.Format("2006-01-02")
	}

	return "backup-" + datePart + ExtensionFor(format)
}

// Result contains archive creation statistics.
type Result struct {
	ArchivePath   string
	FilesArchived int
	TotalSize     int64
	ArchiveSize   int64
}

// CompressionRatio returns the compression ratio as a percentage.
func (r *Result) CompressionRatio() float64 {
	if r.TotalSize == 0 {
		return 100
	}
	return float64(r.ArchiveSize) / float64(r.TotalSize) * 100
}

// Creator handles archive creation.
type Creator struct {
	config    *Config
	outputDir string
}

// NewCreator creates a new archive creator.
func NewCreator(cfg *Config, outputDir string) *Creator {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Creator{
		config:    cfg,
		outputDir: outputDir,
	}
}

// CreateArchive creates an archive from the given files.
// The files map contains source paths as keys and archive paths (relative) as values.
func (c *Creator) CreateArchive(files map[string]string, archiveTime time.Time) (*Result, error) {
	if len(files) == 0 {
		return &Result{}, nil
	}

	format := c.config.Format
	if format == "" {
		format = FormatTarGz
	}

	groupBy := c.config.GroupBy
	if groupBy == "" {
		groupBy = GroupByDaily
	}

	archiveName := GenerateArchiveName(archiveTime, groupBy, format)
	archivePath := filepath.Join(c.outputDir, archiveName)

	// Ensure output directory exists
	if err := os.MkdirAll(c.outputDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create archive directory: %w", err)
	}

	var result *Result
	var err error

	switch format {
	case FormatTar:
		result, err = c.createTarArchive(archivePath, files, false)
	case FormatTarGz:
		result, err = c.createTarArchive(archivePath, files, true)
	case FormatZip:
		result, err = c.createZipArchive(archivePath, files)
	default:
		result, err = c.createTarArchive(archivePath, files, true)
	}

	if err != nil {
		return nil, err
	}

	result.ArchivePath = archivePath
	return result, nil
}

// createTarArchive creates a tar or tar.gz archive.
func (c *Creator) createTarArchive(archivePath string, files map[string]string, compress bool) (*Result, error) {
	file, err := os.Create(archivePath)
	if err != nil {
		return nil, fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	var writer io.WriteCloser = file
	if compress {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	result := &Result{}

	for srcPath, archPath := range files {
		info, err := os.Stat(srcPath)
		if err != nil {
			return nil, fmt.Errorf("stat file %s: %w", srcPath, err)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil, fmt.Errorf("create tar header for %s: %w", srcPath, err)
		}

		// Use the archive path (relative path within archive)
		header.Name = archPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("write tar header for %s: %w", srcPath, err)
		}

		if !info.IsDir() {
			srcFile, err := os.Open(srcPath)
			if err != nil {
				return nil, fmt.Errorf("open file %s: %w", srcPath, err)
			}

			if _, err := io.Copy(tarWriter, srcFile); err != nil {
				srcFile.Close()
				return nil, fmt.Errorf("write file %s to tar: %w", srcPath, err)
			}
			srcFile.Close()

			result.FilesArchived++
			result.TotalSize += info.Size()
		}
	}

	// Close writers to flush data
	tarWriter.Close()
	if compress {
		writer.Close()
	}

	// Get archive size
	archInfo, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("stat archive: %w", err)
	}
	result.ArchiveSize = archInfo.Size()

	return result, nil
}

// createZipArchive creates a zip archive.
func (c *Creator) createZipArchive(archivePath string, files map[string]string) (*Result, error) {
	file, err := os.Create(archivePath)
	if err != nil {
		return nil, fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	result := &Result{}

	for srcPath, archPath := range files {
		info, err := os.Stat(srcPath)
		if err != nil {
			return nil, fmt.Errorf("stat file %s: %w", srcPath, err)
		}

		if info.IsDir() {
			continue
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil, fmt.Errorf("create zip header for %s: %w", srcPath, err)
		}

		// Use the archive path and set compression
		header.Name = archPath
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("create zip entry for %s: %w", srcPath, err)
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			return nil, fmt.Errorf("open file %s: %w", srcPath, err)
		}

		if _, err := io.Copy(writer, srcFile); err != nil {
			srcFile.Close()
			return nil, fmt.Errorf("write file %s to zip: %w", srcPath, err)
		}
		srcFile.Close()

		result.FilesArchived++
		result.TotalSize += info.Size()
	}

	// Close zip writer to flush data
	zipWriter.Close()

	// Get archive size
	archInfo, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("stat archive: %w", err)
	}
	result.ArchiveSize = archInfo.Size()

	return result, nil
}

// ExtractArchive extracts an archive to the given directory.
func ExtractArchive(archivePath, destDir string) error {
	ext := strings.ToLower(filepath.Ext(archivePath))

	// Handle .tar.gz
	if strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz") {
		return extractTarGz(archivePath, destDir)
	}

	switch ext {
	case ".tar":
		return extractTar(archivePath, destDir, false)
	case ".gz":
		return extractTarGz(archivePath, destDir)
	case ".zip":
		return extractZip(archivePath, destDir)
	default:
		return fmt.Errorf("unknown archive format: %s", ext)
	}
}

func extractTar(archivePath, destDir string, compressed bool) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file
	if compressed {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
				return fmt.Errorf("create parent directory for %s: %w", target, err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("create file %s: %w", target, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("write file %s: %w", target, err)
			}
			outFile.Close()
		}
	}

	return nil
}

func extractTarGz(archivePath, destDir string) error {
	return extractTar(archivePath, destDir, true)
}

func extractZip(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		target := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return fmt.Errorf("create directory %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", target, err)
		}

		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", file.Name, err)
		}

		destFile, err := os.Create(target)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("create file %s: %w", target, err)
		}

		if _, err := io.Copy(destFile, srcFile); err != nil {
			srcFile.Close()
			destFile.Close()
			return fmt.Errorf("write file %s: %w", target, err)
		}

		srcFile.Close()
		destFile.Close()
	}

	return nil
}
