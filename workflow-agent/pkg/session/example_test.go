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

package session_test

import (
	"context"
	"fmt"
	"log"
	"seata-go-ai-workflow-agent/pkg/session"
	"seata-go-ai-workflow-agent/pkg/session/config"
	"seata-go-ai-workflow-agent/pkg/session/types"
	"time"
)

func Example_basicUsage() {
	// Initialize with memory store
	cfg := config.DefaultConfig()
	cfg.Store.Type = config.StoreTypeMemory

	if err := session.Initialize(*cfg); err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()

	// Create a new session
	sessionData, err := session.CreateSession(ctx, "user123")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created session: %s\n", sessionData.ID)

	// Add messages to the session
	msg1 := types.NewMessage("", "user", "Hello, AI!")
	if err := session.AddMessage(ctx, sessionData.ID, msg1); err != nil {
		log.Fatal(err)
	}

	msg2 := types.NewMessage("", "assistant", "Hello! How can I help you today?")
	if err := session.AddMessage(ctx, sessionData.ID, msg2); err != nil {
		log.Fatal(err)
	}

	// Retrieve messages
	messages, err := session.GetMessages(ctx, sessionData.ID, 0, 10)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Messages in session: %d\n", len(messages))

	// Output:
	// Created session: sess_[uuid]
	// Messages in session: 2
}

func Example_withCustomOptions() {
	// Initialize with custom configuration
	cfg := config.DefaultConfig()
	cfg.Store.Type = config.StoreTypeMemory
	cfg.Session.DefaultTTL = 30 * time.Minute
	cfg.Session.MaxMessageCount = 100

	if err := session.Initialize(*cfg); err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()

	// Create session with custom options
	sessionData, err := session.CreateSession(ctx, "user456",
		types.WithTTL(time.Hour),
		types.WithMaxMessages(50),
		types.WithMetadata(map[string]string{
			"client_id": "web_app",
			"version":   "1.0.0",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set context data
	if err := session.SetContext(ctx, sessionData.ID, "conversation_type", "technical_support"); err != nil {
		log.Fatal(err)
	}

	// Retrieve context data
	convType, err := session.GetContext(ctx, sessionData.ID, "conversation_type")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Conversation type: %s\n", convType)
	fmt.Printf("Session expires at: %v\n", sessionData.ExpiresAt != nil)

	// Output:
	// Conversation type: technical_support
	// Session expires at: true
}

func Example_multipleStores() {
	// Example showing different store configurations

	// Memory store configuration
	memoryConfig := config.DefaultConfig()
	memoryConfig.Store.Type = config.StoreTypeMemory
	memoryConfig.Store.Memory.MaxSessions = 1000
	memoryConfig.Store.Memory.EvictionPolicy = "lru"

	// Redis store configuration
	redisConfig := config.DefaultConfig()
	redisConfig.Store.Type = config.StoreTypeRedis
	redisConfig.Store.Redis.Addr = "localhost:6379"
	redisConfig.Store.Redis.DB = 0
	redisConfig.Store.Redis.KeyPrefix = "session:"

	// MySQL store configuration
	mysqlConfig := config.DefaultConfig()
	mysqlConfig.Store.Type = config.StoreTypeMySQL
	mysqlConfig.Store.MySQL.Host = "localhost"
	mysqlConfig.Store.MySQL.Port = 3306
	mysqlConfig.Store.MySQL.Database = "sessions"
	mysqlConfig.Store.MySQL.Username = "user"
	mysqlConfig.Store.MySQL.Password = "password"

	fmt.Printf("Memory store configured with max %d sessions\n", memoryConfig.Store.Memory.MaxSessions)
	fmt.Printf("Redis store configured for %s\n", redisConfig.Store.Redis.Addr)
	fmt.Printf("MySQL store configured for %s:%d\n", mysqlConfig.Store.MySQL.Host, mysqlConfig.Store.MySQL.Port)

	// Output:
	// Memory store configured with max 1000 sessions
	// Redis store configured for localhost:6379
	// MySQL store configured for localhost:3306
}
