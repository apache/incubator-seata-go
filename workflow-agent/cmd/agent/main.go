/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"seata-go-ai-workflow-agent/internal/api"
	"seata-go-ai-workflow-agent/internal/config"
	"seata-go-ai-workflow-agent/internal/orchestration"
	"seata-go-ai-workflow-agent/pkg/hubclient"
	"seata-go-ai-workflow-agent/pkg/llm"
	"seata-go-ai-workflow-agent/pkg/logger"
	"seata-go-ai-workflow-agent/pkg/session"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "path to config file")
	port       = flag.Int("port", 8081, "HTTP server port")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg := config.LoadOrDefault(*configPath)

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		OutputPath: cfg.Logger.OutputPath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("=== Workflow Agent Starting ===")

	ctx := context.Background()

	// Initialize OpenAI-compatible provider
	oaiProvider := &openai.OpenAI{
		APIKey: cfg.LLM.APIKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(cfg.LLM.BaseURL),
		},
	}

	// Initialize genkit with OpenAI provider
	g := genkit.Init(ctx, genkit.WithPlugins(oaiProvider))

	// Define custom model
	oaiProvider.DefineModel(cfg.LLM.DefaultModel, ai.ModelOptions{})

	// Build full model name with provider prefix
	fullModelName := "openai/" + cfg.LLM.DefaultModel

	// Initialize LLM client
	llmClient, err := llm.NewClient(&llm.Config{
		Genkit:       g,
		DefaultModel: fullModelName,
		APIKey:       cfg.LLM.APIKey,
		BaseURL:      cfg.LLM.BaseURL,
		Config:       cfg.LLM.Config,
	})
	if err != nil {
		logger.Fatalf("failed to initialize llm client: %v", err)
	}
	logger.Info("LLM client initialized successfully")

	// Initialize Hub client
	hubClient := hubclient.New(&hubclient.Config{
		BaseURL: cfg.Hub.BaseURL,
		Timeout: cfg.GetHubTimeout(),
	})
	logger.Info("Hub client initialized successfully")

	// Test hub connectivity
	if err := hubClient.HealthCheck(ctx); err != nil {
		logger.Warnf("hub health check failed (hub may not be running): %v", err)
		logger.Warnf("Please start agenthub first. Some features may not work.")
	} else {
		logger.Info("Hub is healthy and ready")
	}

	// Initialize session manager
	sessionMgr := session.NewManager(24 * time.Hour) // 24 hour TTL for completed sessions
	logger.Info("Session manager initialized successfully")

	// Initialize orchestrator (legacy synchronous mode)
	orchestrator := orchestration.NewOrchestrator(
		orchestration.OrchestratorConfig{
			MaxRetryCount: cfg.Orchestration.MaxRetryCount,
		},
		llmClient,
		hubClient,
	)
	logger.Info("Orchestrator initialized successfully")

	// Initialize streaming orchestrator (new session-based mode)
	streamingOrchestrator := orchestration.NewStreamingOrchestrator(
		orchestration.OrchestratorConfig{
			MaxRetryCount: cfg.Orchestration.MaxRetryCount,
		},
		llmClient,
		hubClient,
		sessionMgr,
	)
	logger.Info("Streaming orchestrator initialized successfully")

	// Initialize API handlers
	handler := api.NewHandler(orchestrator)
	streamingHandler := api.NewStreamingHandler(streamingOrchestrator)

	// Setup routes
	mux := api.SetupRoutes(handler, streamingHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  180 * time.Second, // Allow long-running orchestration requests
		WriteTimeout: 180 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("HTTP server starting on port %d", *port)
		logger.Infof("API endpoints:")
		logger.Infof("  Legacy synchronous mode:")
		logger.Infof("    POST http://localhost:%d/api/v1/orchestrate - Orchestrate workflow (blocking)", *port)
		logger.Infof("  Session-based streaming mode:")
		logger.Infof("    POST http://localhost:%d/api/v1/sessions - Create new session", *port)
		logger.Infof("    GET  http://localhost:%d/api/v1/sessions - List all sessions", *port)
		logger.Infof("    GET  http://localhost:%d/api/v1/sessions/{id} - Get session status", *port)
		logger.Infof("    GET  http://localhost:%d/api/v1/sessions/{id}/stream - Stream progress (SSE)", *port)
		logger.Infof("    GET  http://localhost:%d/api/v1/sessions/{id}/result - Get final result", *port)
		logger.Infof("  Health check:")
		logger.Infof("    GET  http://localhost:%d/health - Health check", *port)
		logger.Info("=== Workflow Agent Ready ===")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped")
}
