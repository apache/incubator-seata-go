<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

# Session Manager for Genkit Go AI Agent

A comprehensive, high-performance session management package designed for AI agents built with Genkit for Go. Supports multiple storage backends with elegant APIs and production-ready features.

## Features

- **Multiple Storage Backends**: Memory, Redis, and MySQL support
- **Thread-Safe Operations**: Concurrent access with proper synchronization
- **Flexible Configuration**: YAML/JSON config files and environment variables
- **Session Lifecycle Management**: TTL, cleanup, and expiration handling
- **Message Management**: Efficient conversation history storage
- **Context Support**: Key-value context storage per session
- **Health Monitoring**: Store health checks and statistics
- **Graceful Degradation**: Robust error handling and failover

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "seata-go-ai-workflow-agent/pkg/session"
    "seata-go-ai-workflow-agent/pkg/session/config"
    "seata-go-ai-workflow-agent/pkg/session/types"
)

func main() {
    // Initialize with default memory store
    cfg := config.DefaultConfig()
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

    // Add messages
    msg := types.NewMessage("", "user", "Hello, AI!")
    if err := session.AddMessage(ctx, sessionData.ID, msg); err != nil {
        log.Fatal(err)
    }

    // Retrieve messages
    messages, err := session.GetMessages(ctx, sessionData.ID, 0, 10)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Session %s has %d messages", sessionData.ID, len(messages))
}
```

## Configuration

### Using Configuration Files

```yaml
# config.yaml
store:
  type: "redis"
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0
    pool_size: 10
    key_prefix: "session:"

session:
  default_ttl: "24h"
  max_message_count: 1000
  enable_compression: false
  enable_encryption: false

cleanup:
  enabled: true
  interval: "1h"
  batch_size: 100
```

```go
cfg, err := config.LoadFromFile("config.yaml")
if err != nil {
    log.Fatal(err)
}

if err := session.Initialize(*cfg); err != nil {
    log.Fatal(err)
}
```

### Using Environment Variables

```bash
export SESSION_STORE_TYPE=redis
export SESSION_REDIS_ADDR=localhost:6379
export SESSION_REDIS_PASSWORD=secret
export SESSION_DEFAULT_TTL=24h
export SESSION_MAX_MESSAGE_COUNT=1000
```

```go
cfg := config.LoadFromEnv()
if err := session.Initialize(*cfg); err != nil {
    log.Fatal(err)
}
```

## Storage Backends

### Memory Store
Perfect for development and testing:

```go
cfg := config.DefaultConfig()
cfg.Store.Type = config.StoreTypeMemory
cfg.Store.Memory.MaxSessions = 10000
cfg.Store.Memory.EvictionPolicy = "lru"
```

### Redis Store
Recommended for production:

```go
cfg := config.DefaultConfig()
cfg.Store.Type = config.StoreTypeRedis
cfg.Store.Redis.Addr = "localhost:6379"
cfg.Store.Redis.DB = 0
cfg.Store.Redis.PoolSize = 20
```

### MySQL Store
For persistent storage with SQL capabilities:

```go
cfg := config.DefaultConfig()
cfg.Store.Type = config.StoreTypeMySQL
cfg.Store.MySQL.Host = "localhost"
cfg.Store.MySQL.Port = 3306
cfg.Store.MySQL.Database = "sessions"
cfg.Store.MySQL.Username = "user"
cfg.Store.MySQL.Password = "password"
```

## Advanced Usage

### Custom Session Options

```go
sessionData, err := session.CreateSession(ctx, "user123",
    types.WithTTL(2*time.Hour),
    types.WithMaxMessages(500),
    types.WithMetadata(map[string]string{
        "client_type": "mobile",
        "version": "1.2.0",
    }),
    types.WithContext(map[string]interface{}{
        "language": "en",
        "timezone": "UTC",
    }),
)
```

### Context Management

```go
// Set context values
if err := session.SetContext(ctx, sessionID, "user_preferences", userPrefs); err != nil {
    log.Fatal(err)
}

// Get context values
prefs, err := session.GetContext(ctx, sessionID, "user_preferences")
if err != nil {
    log.Fatal(err)
}
```

### Direct Store Usage

```go
// Create store directly
store := store.NewMemoryStore(cfg.Store.Memory)
manager := session.NewManager(store, cfg.Session)

// Start cleanup routine
manager.StartCleanup(time.Hour)
defer manager.Close()
```

## API Reference

### Session Management
- `CreateSession(ctx, userID, ...options) (*SessionData, error)`
- `GetSession(ctx, sessionID) (*SessionData, error)`
- `UpdateSession(ctx, session) error`
- `DeleteSession(ctx, sessionID) error`
- `ListSessions(ctx, userID, offset, limit) ([]*SessionData, error)`

### Message Management
- `AddMessage(ctx, sessionID, message) error`
- `GetMessages(ctx, sessionID, offset, limit) ([]Message, error)`

### Context Management
- `SetContext(ctx, sessionID, key, value) error`
- `GetContext(ctx, sessionID, key) (interface{}, error)`

### Lifecycle Management
- `ExtendTTL(ctx, sessionID, ttl) error`
- `SessionExists(ctx, sessionID) (bool, error)`
- `CountUserSessions(ctx, userID) (int64, error)`

## Best Practices

1. **Always use context**: Pass proper context for cancellation and timeouts
2. **Handle errors gracefully**: Check for specific error types like `ErrSessionNotFound`
3. **Configure appropriate TTLs**: Set reasonable session expiration times
4. **Monitor store health**: Use `Health()` method for monitoring
5. **Use cleanup routines**: Enable automatic cleanup for production
6. **Secure sensitive data**: Use encryption for sensitive session data

## Performance Considerations

- **Memory Store**: Fastest but limited by server memory
- **Redis Store**: Good performance with persistence and clustering support
- **MySQL Store**: Best for complex queries and analytics
- **Connection Pooling**: Configure appropriate pool sizes for database stores
- **Batch Operations**: Use pagination for large result sets

## Error Handling

```go
session, err := session.GetSession(ctx, sessionID)
if err != nil {
    switch {
    case errors.Is(err, types.ErrSessionNotFound):
        // Handle session not found
    case errors.Is(err, types.ErrSessionExpired):
        // Handle expired session
    case errors.Is(err, types.ErrStoreUnavailable):
        // Handle store connectivity issues
    default:
        // Handle other errors
    }
}
```

## Thread Safety

All operations are thread-safe and can be used concurrently from multiple goroutines. The package uses appropriate synchronization mechanisms internally.

## License

This package is part of the seata-go-ai-workflow-agent project.