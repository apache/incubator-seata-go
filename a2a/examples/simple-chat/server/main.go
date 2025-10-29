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
	"log"
	"os"
	"os/signal"
	"syscall"

	"seata-go-ai-a2a/pkg/a2a"
)

func main() {
	log.Println("Starting Simple Chat A2A Server...")

	// Create message handler and agent card provider
	messageHandler := &EchoMessageHandler{}
	cardProvider := NewChatAgentCardProvider()

	// Get the base agent card for server initialization
	baseCard, err := cardProvider.GetAgentCard(context.Background())
	if err != nil {
		log.Fatalf("Failed to get base agent card: %v", err)
	}

	// Create A2A server with handlers
	server, err := a2a.NewServerWithHandlers(baseCard, messageHandler, cardProvider)
	if err != nil {
		log.Fatalf("Failed to create A2A server: %v", err)
	}

	// Start the server
	log.Println("Starting server on :8080 (HTTP/REST/JSON-RPC) and :9090 (gRPC)...")
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("✅ Server started successfully!")
	log.Println("Available endpoints:")
	log.Println("  - gRPC: localhost:9090")
	log.Println("  - HTTP REST: http://localhost:8080/api/v1/")
	log.Println("  - JSON-RPC: http://localhost:8080/jsonrpc")
	log.Println("  - Agent Card: http://localhost:8080/api/v1/agent-card")
	log.Println("")
	log.Println("Try these example requests:")
	log.Println("  curl -X POST http://localhost:8080/api/v1/messages -H 'Content-Type: application/json' -d '{\"message\":{\"role\":1,\"parts\":[{\"type\":\"text\",\"data\":{\"text\":\"Hello!\"}}]}}'")
	log.Println("  curl http://localhost:8080/api/v1/agent-card")
	log.Println("")
	log.Println("Press Ctrl+C to stop the server")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down server...")
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("Server stopped")
}
