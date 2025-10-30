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
	"os"
	"os/signal"
	"syscall"

	"seata-go-ai-txn-agent/pkg/agent"
	"seata-go-ai-txn-agent/pkg/trans"
	"seata-go-ai-txn-agent/pkg/utils"
)

const (
	defaultPort     = "8080"
	defaultLogLevel = "info"
	defaultModel    = "qwen-max"
	defaultBaseURL  = "https://dashscope.aliyuncs.com/compatible-mode/v1"
)

func main() {
	// Command line flags
	var (
		port       = flag.String("port", defaultPort, "Server port")
		apiKey     = flag.String("api-key", "", "LLM API key")
		baseURL    = flag.String("base-url", defaultBaseURL, "LLM API base URL")
		model      = flag.String("model", defaultModel, "LLM model name")
		logLevel   = flag.String("log-level", defaultLogLevel, "Log level (debug, info, warn, error)")
		timeout    = flag.Int("timeout", 120, "LLM request timeout in seconds")
		maxRetries = flag.Int("max-retries", 3, "Maximum number of retries for LLM requests")
	)
	flag.Parse()

	// Initialize logger
	logger := utils.GetLogger("main")
	logger.Info("Starting Seata Go Workflow Agent WebSocket Server")

	// Validate required parameters
	if *apiKey == "" {
		// Try to get from environment
		if envKey := os.Getenv("API_KEY"); envKey != "" {
			*apiKey = envKey
		} else {
			logger.Fatal("API key is required (use --api-key or API_KEY env var)")
		}
	}

	// Configure agent
	agentConfig := agent.AgentConfig{
		APIKey:     *apiKey,
		BaseURL:    *baseURL,
		Model:      *model,
		Timeout:    *timeout,
		MaxRetries: *maxRetries,
	}

	logger.Debug("Agent configuration ready")

	// Create and start server
	server := trans.NewSimpleServer(*port, agentConfig)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.WithField("signal", sig).Info("Received shutdown signal")
		cancel()
	}()

	// Print startup information
	printStartupInfo(*port, *model, *baseURL, *logLevel)

	// Start server
	if err := server.Start(ctx); err != nil {
		logger.WithError(err).Fatal("Server failed to start")
	}

	logger.Info("Server stopped")
}

func printStartupInfo(port, model, baseURL, logLevel string) {
	fmt.Printf(`
🚀 Seata Go Workflow Agent WebSocket Server (Simple Mode)

📊 Server Information:
   • Port: %s
   • Model: %s
   • API URL: %s
   • Log Level: %s

🌐 Endpoints:
   • WebSocket: ws://localhost:%s/ws
   • Health: http://localhost:%s/health
   • Status: http://localhost:%s/api/v1/status

💡 Usage:
   • Connect your frontend to ws://localhost:%s/ws
   • Send JSON messages with type 'user_input'
   • Press Ctrl+C to stop the server

`, port, model, baseURL, logLevel, port, port, port, port)
}
