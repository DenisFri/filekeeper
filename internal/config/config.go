package config

import (
	"encoding/json"
	"os"
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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	cfg := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
