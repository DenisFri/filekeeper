package main

import (
	"filekeeper/internal/backup"
	"filekeeper/internal/config"
	"fmt"
	"time"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Run the service
	for {
		err := backup.RunBackup(cfg)
		if err != nil {
			fmt.Printf("Error running backup: %v\n", err)
		}
		time.Sleep(time.Duration(cfg.RunInterval) * time.Second)
	}
}
