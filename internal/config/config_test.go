package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_PruneAfterHours(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		hours   float32
		wantErr bool
	}{
		{"positive hours", 24.0, false},
		{"zero hours", 0, true},
		{"negative hours", -1, true},
		{"small positive", 0.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PruneAfterHours: tt.hours,
				RunInterval:     3600,
				TargetFolder:    tempDir,
				EnableBackup:    false,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_RunInterval(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		interval int
		wantErr  bool
	}{
		{"positive interval", 3600, false},
		{"zero interval", 0, true},
		{"negative interval", -1, true},
		{"small positive", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PruneAfterHours: 24,
				RunInterval:     tt.interval,
				TargetFolder:    tempDir,
				EnableBackup:    false,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_TargetFolder(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		targetFolder string
		wantErr      bool
	}{
		{"existing directory", tempDir, false},
		{"empty path", "", true},
		{"non-existent path", "/non/existent/path/12345", true},
		{"file instead of directory", tempFile, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PruneAfterHours: 24,
				RunInterval:     3600,
				TargetFolder:    tt.targetFolder,
				EnableBackup:    false,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_BackupPath(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		enableBackup bool
		backupPath   string
		wantErr      bool
	}{
		{"backup disabled, empty path", false, "", false},
		{"backup disabled, with path", false, tempDir, false},
		{"backup enabled, valid path", true, tempDir, false},
		{"backup enabled, empty path", true, "", true},
		{"backup enabled, file as path", true, tempFile, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PruneAfterHours: 24,
				RunInterval:     3600,
				TargetFolder:    tempDir,
				EnableBackup:    tt.enableBackup,
				BackupPath:      tt.backupPath,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_RemoteBackup(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		remoteBackup string
		wantErr      bool
	}{
		{"empty (disabled)", "", false},
		{"valid user@host:/path", "user@host.example.com:/backup", false},
		{"valid host:/path", "host.example.com:/backup", false},
		{"valid with underscores", "user_name@host-name.example.com:/path/to/backup", false},
		{"invalid no colon", "user@host/path", true},
		{"invalid no path", "user@host:", true},
		{"invalid just path", "/local/path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PruneAfterHours: 24,
				RunInterval:     3600,
				TargetFolder:    tempDir,
				EnableBackup:    false,
				RemoteBackup:    tt.remoteBackup,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_WithValidation(t *testing.T) {
	tempDir := t.TempDir()

	// Create a valid config file
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"prune_after_hours": 24,
		"target_folder": "` + filepath.ToSlash(tempDir) + `",
		"run_interval": 3600,
		"backup_path": "` + filepath.ToSlash(tempDir) + `",
		"remote_backup": "",
		"enable_backup": false
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.PruneAfterHours != 24 {
		t.Errorf("PruneAfterHours = %f, want 24", cfg.PruneAfterHours)
	}
}

func TestLoadConfig_InvalidConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Create an invalid config file (negative prune_after_hours)
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"prune_after_hours": -1,
		"target_folder": "` + filepath.ToSlash(tempDir) + `",
		"run_interval": 3600,
		"backup_path": "",
		"remote_backup": "",
		"enable_backup": false
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() expected error for invalid config, got nil")
	}
}
