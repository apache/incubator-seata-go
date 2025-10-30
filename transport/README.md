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

# Seata Go AI Transport Layer Implementation

This package provides a unified transport layer for both JSON-RPC and gRPC protocols, supporting both streaming and non-streaming operations.

## Features

### Protocols Supported
- **JSON-RPC 2.0**: HTTP-based with Server-Sent Events (SSE) for streaming
- **gRPC**: Full support including server streaming RPCs

### Core Capabilities
- **Unified Interface**: Common abstractions for both protocols
- **Streaming Support**: 
  - JSON-RPC: SSE-based streaming
  - gRPC: Server streaming RPCs
- **Registry Pattern**: Automatic protocol detection and builder registration
- **Type Safety**: Proper interface contracts with compile-time checks
- **Error Handling**: Protocol-specific error types with common interface
- **Extensible**: Easy to add new transport protocols

## Architecture

### Package Structure
```
pkg/transport/
├── common/          # Common interfaces and utilities
│   ├── types.go     # Core types and interfaces
│   ├── registry.go  # Protocol registry and factory
│   ├── utils.go     # Utility functions
│   └── errors.go    # Error definitions
├── jsonrpc/         # JSON-RPC implementation
│   ├── types.go     # JSON-RPC specific types
│   ├── client.go    # HTTP client implementation
│   ├── server.go    # HTTP server implementation
│   ├── sse_*.go     # Server-Sent Events streaming
│   ├── builder.go   # Factory builders
│   └── jsonrpc.go   # Package exports and registration
├── grpc/            # gRPC implementation
│   ├── client.go    # gRPC client wrapper
│   ├── server.go    # gRPC server wrapper
│   ├── stream_*.go  # Streaming implementations
│   ├── builder.go   # Factory builders
│   └── grpc.go      # Package exports and registration
└── transport.go     # Main package exports
```

### Core Interfaces

#### Client Interface
```go
type Client interface {
    Call(ctx context.Context, msg *Message) (*Response, error)
    Stream(ctx context.Context, msg *Message) (StreamReader, error)
    Protocol() Protocol
    Close() error
}
```

#### Server Interface
```go
type Server interface {
    Serve() error
    Stop(ctx context.Context) error
    RegisterHandler(method string, handler Handler)
    RegisterStreamHandler(method string, handler StreamHandler)
    Protocol() Protocol
}
```

## Usage Examples

### Basic JSON-RPC Server
```go
import "seata-go-ai-transport"

// Create server
config := &transport.Config{
    Protocol: transport.ProtocolJSONRPC,
    Address:  ":8080",
    Options: map[string]interface{}{
        "streaming": true, // Enable SSE
    },
}

server, err := transport.CreateServer(config)
if err != nil {
    log.Fatal(err)
}

// Register handler
server.RegisterHandler("echo", func(ctx context.Context, req *transport.Message) (*transport.Response, error) {
    return transport.CreateResponse(map[string]interface{}{
        "echo": string(req.Payload),
    })
})

// Start server
go server.Serve()
```

### Basic JSON-RPC Client
```go
// Create client
config := &transport.Config{
    Protocol: transport.ProtocolJSONRPC,
    Address:  "http://localhost:8080",
    Options: map[string]interface{}{
        "streaming": true,
    },
}

client, err := transport.CreateClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Make call
msg, _ := transport.MarshalMessage(map[string]string{"hello": "world"})
msg.Method = "echo"

resp, err := client.Call(context.Background(), msg)
if err != nil {
    log.Fatal(err)
}

var result map[string]interface{}
transport.UnmarshalResponse(resp, &result)
fmt.Println(result)
```

### Streaming Example
```go
// Register streaming handler
server.RegisterStreamHandler("stream", func(ctx context.Context, req *transport.Message, stream transport.StreamWriter) error {
    for i := 0; i < 5; i++ {
        resp, _ := transport.CreateStreamResponse(map[string]int{"count": i}, i == 4)
        if err := stream.Send(resp); err != nil {
            return err
        }
        time.Sleep(time.Second)
    }
    return nil
})

// Client streaming call
streamMsg, _ := transport.MarshalMessage(nil)
streamMsg.Method = "stream"

stream, err := client.Stream(context.Background(), streamMsg)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    resp, err := stream.Recv()
    if err != nil {
        break
    }
    if resp.Done {
        break
    }
    fmt.Printf("Stream data: %s\n", string(resp.Data))
}
```

### gRPC Usage
```go
// Create gRPC server
config := &transport.Config{
    Protocol: transport.ProtocolGRPC,
    Address:  ":9090",
    Options: map[string]interface{}{
        "insecure": true,
    },
}

server, err := transport.CreateServer(config)
if err != nil {
    log.Fatal(err)
}

// Register methods (requires proto definitions)
grpcServer := server.(*grpc.Server)
grpcServer.RegisterMethod("Ping", grpc.CreateUnaryMethodInfo(
    "/service/Ping",
    &PingRequest{},
    &PongResponse{},
))

// Start server
go server.Serve()
```

### Direct Usage (Without Registry)
```go
import (
    "seata-go-ai-transport/jsonrpc"
    "seata-go-ai-transport/grpc"
)

// Direct JSON-RPC client
jsonClient := jsonrpc.NewSSEJSONRPCClient("http://localhost:8080", "")
defer jsonClient.Close()

// Direct gRPC client
grpcClient, err := grpc.NewGRPCClient("localhost:9090", true)
if err != nil {
    log.Fatal(err)
}
defer grpcClient.Close()
```

## Configuration Options

### JSON-RPC Options
```go
Options: map[string]interface{}{
    "endpoint":        "http://localhost:8080",    // Server endpoint
    "stream_endpoint": "http://localhost:8080/stream", // SSE endpoint
    "timeout":         "30s",                      // Request timeout
    "headers": map[string]string{                  // Custom headers
        "Authorization": "Bearer token",
    },
    "streaming":       true,                       // Enable SSE streaming
    "stream_path":     "/stream",                  // SSE path for server
    "read_timeout":    "30s",                      // Server read timeout
    "write_timeout":   "30s",                      // Server write timeout
    "idle_timeout":    "120s",                     // Server idle timeout
}
```

### gRPC Options
```go
Options: map[string]interface{}{
    "target":          "localhost:9090",          // Server target
    "insecure":        true,                      // Use insecure connection
    "timeout":         "30s",                     // Connection timeout
    "dial_options":    []grpc.DialOption{...},    // Custom dial options
    "server_options":  []grpc.ServerOption{...},  // Custom server options
    "methods": map[string]grpc.MethodInfo{        // Method definitions
        "Ping": grpc.CreateUnaryMethodInfo(...),
    },
}
```

## Error Handling

The transport layer provides structured error handling:

```go
// Check if error is transport-specific
if transport.IsTransportError(err) {
    if transportErr, ok := transport.GetTransportError(err); ok {
        fmt.Printf("Protocol: %s, Code: %d, Message: %s\n", 
            transportErr.Protocol, transportErr.Code, transportErr.Message)
    }
}

// JSON-RPC specific errors
if jsonrpc.IsJSONRPCError(err) {
    if jsonErr, ok := jsonrpc.GetJSONRPCError(err); ok {
        fmt.Printf("JSON-RPC Error: %d - %s\n", jsonErr.Code, jsonErr.Message)
    }
}
```

## Implementation Details

### JSON-RPC Features
- **HTTP/1.1 Transport**: Standard HTTP POST requests
- **SSE Streaming**: Server-Sent Events for real-time streaming
- **JSON-RPC 2.0 Compliance**: Full specification compliance
- **Error Propagation**: Standard JSON-RPC error codes
- **Content Negotiation**: Proper HTTP headers and content types

### gRPC Features
- **Server Streaming**: Full server streaming RPC support
- **Protocol Buffers**: Integration with existing protobuf definitions
- **Connection Management**: Proper connection lifecycle
- **Metadata Support**: gRPC metadata as transport headers
- **Graceful Shutdown**: Clean server shutdown with context cancellation

### Streaming Implementations
- **JSON-RPC**: Uses SSE (Server-Sent Events) over HTTP
- **gRPC**: Uses native gRPC server streaming RPCs
- **Common Interface**: Both implement the same `StreamReader`/`StreamWriter` interfaces
- **Error Handling**: Stream-specific error propagation

## Thread Safety

All implementations are thread-safe:
- **Clients**: Safe for concurrent use
- **Servers**: Handle multiple concurrent connections
- **Streams**: Individual streams are not thread-safe (by design)
- **Registry**: Thread-safe protocol registration and lookup

## Extension Points

To add a new transport protocol:

1. Implement the `Client` and `Server` interfaces
2. Create `ClientBuilder` and `ServerBuilder` implementations
3. Register builders in package `init()` function
4. Add protocol constant to `common/types.go`

Example:
```go
func init() {
    common.RegisterClientBuilder(common.ProtocolWebSocket, &WSClientBuilder{})
    common.RegisterServerBuilder(common.ProtocolWebSocket, &WSServerBuilder{})
}
```