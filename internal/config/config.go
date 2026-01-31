package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

type Config struct {
	PruneAfterHours float32 `json:"prune_after_hours"`
	TargetFolder    string  `json:"target_folder"`
	RunInterval     int     `json:"run_interval"`
	BackupPath      string  `json:"backup_path"`
	RemoteBackup    string  `json:"remote_backup"`
	EnableBackup    bool    `json:"enable_backup"`
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
		if c.BackupPath == "" {
			return fmt.Errorf("backup_path is required when enable_backup is true")
		}

		// Check if backup path's parent directory exists (backup dir will be created if needed)
		parentDir := c.BackupPath
		if info, err := os.Stat(parentDir); err == nil && !info.IsDir() {
			return fmt.Errorf("backup_path exists but is not a directory: %s", c.BackupPath)
		}
	}

	// Validate remote backup format if specified (user@host:/path or host:/path)
	if c.RemoteBackup != "" {
		remotePattern := regexp.MustCompile(`^([a-zA-Z0-9._-]+@)?[a-zA-Z0-9._-]+:.+$`)
		if !remotePattern.MatchString(c.RemoteBackup) {
			return fmt.Errorf("remote_backup has invalid format, expected user@host:/path or host:/path, got: %s", c.RemoteBackup)
		}
	}

	return nil
}
