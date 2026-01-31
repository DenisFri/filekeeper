package main

import (
	"filekeeper/internal/backup"
	"filekeeper/internal/config"
	"filekeeper/internal/logger"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, cfg.LogFormat)

	log.Info("filekeeper started",
		slog.Float64("prune_after_hours", float64(cfg.PruneAfterHours)),
		slog.Int("run_interval_seconds", cfg.RunInterval),
		slog.String("target_folder", cfg.TargetFolder),
		slog.Bool("backup_enabled", cfg.EnableBackup),
	)

	// Run the service
	for {
		err := backup.RunBackup(cfg, log)
		if err != nil {
			log.Error("backup cycle failed", slog.String("error", err.Error()))
		}
		time.Sleep(time.Duration(cfg.RunInterval) * time.Second)
	}
}
