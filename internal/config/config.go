package config

import (
	"encoding/json"
	"filekeeper/internal/archive"
	"filekeeper/pkg/compression"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CompressionConfig holds compression settings for backups.
type CompressionConfig struct {
	Enabled   bool   `json:"enabled"`   // Enable compression for backups
	Algorithm string `json:"algorithm"` // Compression algorithm: "none", "gzip"
	Level     int    `json:"level"`     // Compression level (gzip: 1-9, default: 6)
}

// ArchiveConfig holds archive mode settings for backups.
type ArchiveConfig struct {
	Enabled bool   `json:"enabled"`  // Enable archive mode (bundle files into single archive)
	Format  string `json:"format"`   // Archive format: "tar", "tar.gz", "zip"
	GroupBy string `json:"group_by"` // Group files by: "daily", "weekly", "monthly"
}

type Config struct {
	PruneAfterHours       float32            `json:"prune_after_hours"`
	TargetFolder          string             `json:"target_folder"`
	RunInterval           int                `json:"run_interval"`
	BackupPath            string             `json:"backup_path"`              // Single backup path (backward compatible)
	BackupPaths           []string           `json:"backup_paths"`             // Multiple backup paths
	RemoteBackup          string             `json:"remote_backup"`            // Single remote backup (backward compatible)
	RemoteBackups         []string           `json:"remote_backups"`           // Multiple remote backups
	EnableBackup          bool               `json:"enable_backup"`
	LogLevel              string             `json:"log_level"`                // debug, info, warn, error (default: info)
	LogFormat             string             `json:"log_format"`               // text, json (default: text)
	ErrorThresholdPercent float64            `json:"error_threshold_percent"`  // max failure rate before stopping (0-100, default: 0 = disabled)
	Compression           *CompressionConfig `json:"compression,omitempty"`    // Compression settings for backups
	Archive               *ArchiveConfig     `json:"archive,omitempty"`        // Archive mode settings for backups
}

// GetCompressionConfig returns the compression configuration, converting to the pkg format.
func (c *Config) GetCompressionConfig() *compression.Config {
	if c.Compression == nil || !c.Compression.Enabled {
		return &compression.Config{Enabled: false}
	}

	alg := compression.Algorithm(strings.ToLower(c.Compression.Algorithm))
	if alg == "" {
		alg = compression.Gzip // Default to gzip if enabled but no algorithm specified
	}

	level := c.Compression.Level
	if level == 0 {
		level = 6 // Default compression level
	}

	return &compression.Config{
		Enabled:   true,
		Algorithm: alg,
		Level:     level,
	}
}

// GetArchiveConfig returns the archive configuration, converting to the internal/archive format.
func (c *Config) GetArchiveConfig() *archive.Config {
	if c.Archive == nil || !c.Archive.Enabled {
		return &archive.Config{Enabled: false}
	}

	format := archive.Format(strings.ToLower(c.Archive.Format))
	if format == "" {
		format = archive.FormatTarGz // Default to tar.gz if enabled but no format specified
	}

	groupBy := archive.GroupBy(strings.ToLower(c.Archive.GroupBy))
	if groupBy == "" {
		groupBy = archive.GroupByDaily // Default to daily grouping
	}

	return &archive.Config{
		Enabled: true,
		Format:  format,
		GroupBy: groupBy,
	}
}

// GetBackupPaths returns all configured backup paths, merging single and multiple path configs.
func (c *Config) GetBackupPaths() []string {
	paths := make([]string, 0)

	// Add single backup_path if set
	if c.BackupPath != "" {
		paths = append(paths, c.BackupPath)
	}

	// Add all backup_paths
	for _, p := range c.BackupPaths {
		if p != "" {
			// Avoid duplicates
			found := false
			for _, existing := range paths {
				if existing == p {
					found = true
					break
				}
			}
			if !found {
				paths = append(paths, p)
			}
		}
	}

	return paths
}

// GetRemoteBackups returns all configured remote backup destinations.
func (c *Config) GetRemoteBackups() []string {
	remotes := make([]string, 0)

	// Add single remote_backup if set
	if c.RemoteBackup != "" {
		remotes = append(remotes, c.RemoteBackup)
	}

	// Add all remote_backups
	for _, r := range c.RemoteBackups {
		if r != "" {
			// Avoid duplicates
			found := false
			for _, existing := range remotes {
				if existing == r {
					found = true
					break
				}
			}
			if !found {
				remotes = append(remotes, r)
			}
		}
	}

	return remotes
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(cfg)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks that all configuration values are valid and safe to use.
func (c *Config) Validate() error {
	if c.PruneAfterHours <= 0 {
		return fmt.Errorf("prune_after_hours must be positive, got %f", c.PruneAfterHours)
	}

	if c.RunInterval <= 0 {
		return fmt.Errorf("run_interval must be positive, got %d", c.RunInterval)
	}

	if c.TargetFolder == "" {
		return fmt.Errorf("target_folder is required")
	}

	// Check if target folder exists
	info, err := os.Stat(c.TargetFolder)
	if os.IsNotExist(err) {
		return fmt.Errorf("target_folder does not exist: %s", c.TargetFolder)
	}
	if err != nil {
		return fmt.Errorf("cannot access target_folder: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target_folder is not a directory: %s", c.TargetFolder)
	}

	// Validate backup settings
	if c.EnableBackup {
		backupPaths := c.GetBackupPaths()
		if len(backupPaths) == 0 {
			return fmt.Errorf("at least one backup_path or backup_paths entry is required when enable_backup is true")
		}

		// Validate each backup path
		for _, path := range backupPaths {
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return fmt.Errorf("backup path exists but is not a directory: %s", path)
			}
		}
	}

	// Validate remote backup format if specified (user@host:/path or host:/path)
	remotePattern := regexp.MustCompile(`^([a-zA-Z0-9._-]+@)?[a-zA-Z0-9._-]+:.+$`)

	if c.RemoteBackup != "" {
		if !remotePattern.MatchString(c.RemoteBackup) {
			return fmt.Errorf("remote_backup has invalid format, expected user@host:/path or host:/path, got: %s", c.RemoteBackup)
		}
	}

	// Validate all remote_backups entries
	for _, remote := range c.RemoteBackups {
		if remote != "" && !remotePattern.MatchString(remote) {
			return fmt.Errorf("remote_backups entry has invalid format, expected user@host:/path or host:/path, got: %s", remote)
		}
	}

	// Validate log level if specified
	if c.LogLevel != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		level := strings.ToLower(c.LogLevel)
		valid := false
		for _, v := range validLevels {
			if level == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("log_level must be one of: debug, info, warn, error; got: %s", c.LogLevel)
		}
	}

	// Validate log format if specified
	if c.LogFormat != "" {
		format := strings.ToLower(c.LogFormat)
		if format != "text" && format != "json" {
			return fmt.Errorf("log_format must be 'text' or 'json'; got: %s", c.LogFormat)
		}
	}

	// Validate error threshold percent
	if c.ErrorThresholdPercent < 0 || c.ErrorThresholdPercent > 100 {
		return fmt.Errorf("error_threshold_percent must be between 0 and 100, got: %f", c.ErrorThresholdPercent)
	}

	// Validate compression settings
	if c.Compression != nil && c.Compression.Enabled {
		compressionCfg := c.GetCompressionConfig()
		if err := compressionCfg.Validate(); err != nil {
			return fmt.Errorf("compression: %w", err)
		}
	}

	// Validate archive settings
	if c.Archive != nil && c.Archive.Enabled {
		archiveCfg := c.GetArchiveConfig()
		if err := archiveCfg.Validate(); err != nil {
			return fmt.Errorf("archive: %w", err)
		}

		// Archive mode and per-file compression are mutually exclusive
		if c.Compression != nil && c.Compression.Enabled {
			return fmt.Errorf("archive mode and compression cannot be enabled at the same time; use archive format 'tar.gz' for compressed archives")
		}
	}

	return nil
}
