package main

import (
	"log"
	"os"

	"agenthub/internal/app"
)

func main() {
	// Get config path from command line args or use default
	configPath := "config.yaml"
	if len(os.Args) > 2 && os.Args[1] == "--config" {
		configPath = os.Args[2]
	} else if len(os.Args) > 1 && os.Args[1] != "--config" {
		configPath = os.Args[1]
	}

	// Create and run application
	application, err := app.NewApplication(app.ApplicationConfig{
		ConfigPath: configPath,
	})
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
