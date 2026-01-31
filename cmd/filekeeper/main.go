package main

import (
	"context"
	"filekeeper/internal/backup"
	"filekeeper/internal/config"
	"filekeeper/internal/logger"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		log.Info("shutdown signal received",
			slog.String("signal", sig.String()),
		)
		log.Info("finishing current operation, please wait...")
		cancel()
	}()

	// Run the service
	for {
		select {
		case <-ctx.Done():
			log.Info("shutdown complete")
			return
		default:
			err := backup.RunBackup(ctx, cfg, log)
			if err != nil {
				// Don't log context cancellation as an error
				if ctx.Err() != nil {
					log.Info("backup interrupted by shutdown")
				} else {
					log.Error("backup cycle failed", slog.String("error", err.Error()))
				}
			}

			// Check if shutdown was requested before sleeping
			select {
			case <-ctx.Done():
				log.Info("shutdown complete")
				return
			case <-time.After(time.Duration(cfg.RunInterval) * time.Second):
				// Continue to next iteration
			}
		}
	}
}
