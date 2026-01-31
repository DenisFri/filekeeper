package main

import (
	"context"
	"filekeeper/internal/backup"
	"filekeeper/internal/config"
	"filekeeper/internal/logger"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Version information - set by build system (e.g., goreleaser)
var (
	Version   = "dev"
	BuildDate = "unknown"
	Commit    = "unknown"
)

func main() {
	// Define flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.StringVar(configPath, "c", "config.json", "Path to configuration file (shorthand)")

	once := flag.Bool("once", false, "Run once and exit (no loop)")
	flag.BoolVar(once, "1", false, "Run once and exit (shorthand)")

	dryRun := flag.Bool("dry-run", false, "Show what would be done without doing it")
	flag.BoolVar(dryRun, "n", false, "Show what would be done (shorthand)")

	verbose := flag.Bool("verbose", false, "Enable verbose/debug logging")
	flag.BoolVar(verbose, "v", false, "Enable verbose logging (shorthand)")

	version := flag.Bool("version", false, "Show version and exit")
	flag.BoolVar(version, "V", false, "Show version and exit (shorthand)")

	validate := flag.Bool("validate", false, "Validate configuration and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Filekeeper - Automatic file backup and pruning service\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --config string    Path to configuration file (default \"config.json\")\n")
		fmt.Fprintf(os.Stderr, "  -1, --once             Run once and exit (no loop)\n")
		fmt.Fprintf(os.Stderr, "  -n, --dry-run          Show what would be done without doing it\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose          Enable verbose/debug logging\n")
		fmt.Fprintf(os.Stderr, "  -V, --version          Show version and exit\n")
		fmt.Fprintf(os.Stderr, "      --validate         Validate configuration and exit\n")
		fmt.Fprintf(os.Stderr, "  -h, --help             Show this help message\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --config /etc/filekeeper/config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --once --dry-run\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --validate --config new-config.json\n", os.Args[0])
	}

	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("filekeeper %s (built %s, commit %s)\n", Version, BuildDate, Commit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Handle validate flag
	if *validate {
		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Configuration invalid: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Configuration is valid")
		os.Exit(0)
	}

	// Override log level if verbose
	if *verbose {
		cfg.LogLevel = "debug"
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, cfg.LogFormat)

	if *dryRun {
		log.Info("running in dry-run mode - no changes will be made")
	}

	log.Info("filekeeper started",
		slog.String("version", Version),
		slog.Float64("prune_after_hours", float64(cfg.PruneAfterHours)),
		slog.Int("run_interval_seconds", cfg.RunInterval),
		slog.String("target_folder", cfg.TargetFolder),
		slog.Bool("backup_enabled", cfg.EnableBackup),
		slog.Bool("dry_run", *dryRun),
		slog.Bool("once", *once),
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

	// Create run options
	opts := &backup.RunOptions{
		DryRun: *dryRun,
	}

	// Run the service
	for {
		select {
		case <-ctx.Done():
			log.Info("shutdown complete")
			return
		default:
			result, err := backup.RunBackup(ctx, cfg, opts, log)

			// Log result summary
			if result != nil {
				if result.HasErrors() {
					log.Warn("backup cycle completed with errors",
						slog.Int("succeeded", result.Succeeded),
						slog.Int("failed", result.Failed),
						slog.Int("backed_up", result.BackedUp),
						slog.Int("pruned", result.Pruned),
						slog.Float64("failure_rate_percent", result.FailureRate()),
					)
				} else if result.Succeeded > 0 || result.Pruned > 0 {
					log.Info("backup cycle completed",
						slog.Int("succeeded", result.Succeeded),
						slog.Int("backed_up", result.BackedUp),
						slog.Int("pruned", result.Pruned),
						slog.Int64("total_bytes", result.TotalBytes),
					)
				}
			}

			if err != nil {
				// Don't log context cancellation as an error
				if ctx.Err() != nil {
					log.Info("backup interrupted by shutdown")
				} else {
					log.Error("backup cycle failed", slog.String("error", err.Error()))
				}
			}

			// If running once, exit after first cycle
			if *once {
				if err != nil && ctx.Err() == nil {
					os.Exit(1)
				}
				log.Info("single run complete, exiting")
				return
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
