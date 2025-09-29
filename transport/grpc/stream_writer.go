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

package grpc

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"seata-go-ai-transport/common"
)

// GRPCStreamWriter implements common.StreamWriter for gRPC streams
type GRPCStreamWriter struct {
	stream     grpc.ServerStream
	outputType proto.Message
	closed     int32
	mu         sync.Mutex
}

var _ common.StreamWriter = (*GRPCStreamWriter)(nil)

// NewGRPCStreamWriter creates a new gRPC stream writer
func NewGRPCStreamWriter(stream grpc.ServerStream, outputType proto.Message) *GRPCStreamWriter {
	return &GRPCStreamWriter{
		stream:     stream,
		outputType: outputType,
	}
}

// Send sends a message to the stream
func (w *GRPCStreamWriter) Send(resp *common.StreamResponse) error {
	if atomic.LoadInt32(&w.closed) == 1 {
		return common.ErrStreamClosed
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Handle error response
	if resp.Error != nil {
		return resp.Error
	}

	// If marked as done with no data, don't send anything
	if resp.Done && len(resp.Data) == 0 {
		return nil
	}

	// Create output message
	output := proto.Clone(w.outputType)

	// Unmarshal response data if present
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, output); err != nil {
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	// Send message
	if err := w.stream.SendMsg(output); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Close closes the stream
func (w *GRPCStreamWriter) Close() error {
	if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		return common.ErrStreamClosed
	}

	// gRPC server streams are automatically closed when the handler returns
	// No explicit close action needed
	return nil
}

// SendError sends an error through the stream
func (w *GRPCStreamWriter) SendError(err error) error {
	return w.Send(&common.StreamResponse{
		Error: err,
		Done:  true,
	})
}

// SendData sends data through the stream
func (w *GRPCStreamWriter) SendData(data interface{}, done bool) error {
	var jsonData []byte
	var err error

	if data != nil {
		jsonData, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
	}

	return w.Send(&common.StreamResponse{
		Data: jsonData,
		Done: done,
	})
}
