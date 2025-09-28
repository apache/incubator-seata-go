package main

import (
	"log"

	"test-provider/internal/agent"
	"test-provider/internal/config"
	"test-provider/internal/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := utils.NewLogger(cfg.Log.Level)
	defer logger.Sync()

	agent := agent.New(cfg, logger)
	if err := agent.Start(); err != nil {
		logger.Fatal("Failed to start agent", "error", err)
	}
}
