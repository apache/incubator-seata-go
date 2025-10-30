package main

import (
	"log"

	"test-agent/internal/agent"
	"test-agent/internal/config"
	"test-agent/internal/utils"
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
