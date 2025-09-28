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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"seata-go-ai-transport/common"
)

// GRPCStreamReader implements common.StreamReader for gRPC streams
type GRPCStreamReader struct {
	ctx        context.Context
	stream     grpc.ClientStream
	outputType proto.Message
	closed     int32
}

var _ common.StreamReader = (*GRPCStreamReader)(nil)

// NewGRPCStreamReader creates a new gRPC stream reader
func NewGRPCStreamReader(ctx context.Context, stream grpc.ClientStream, outputType proto.Message) *GRPCStreamReader {
	return &GRPCStreamReader{
		ctx:        ctx,
		stream:     stream,
		outputType: outputType,
	}
}

// Recv receives the next message from the stream
func (r *GRPCStreamReader) Recv() (*common.StreamResponse, error) {
	if atomic.LoadInt32(&r.closed) == 1 {
		return nil, common.ErrStreamClosed
	}

	// Create output message instance
	output := proto.Clone(r.outputType)

	// Receive message from stream
	err := r.stream.RecvMsg(output)
	if err != nil {
		if err == io.EOF {
			return &common.StreamResponse{
				Done: true,
			}, nil
		}
		return &common.StreamResponse{
			Error: err,
			Done:  true,
		}, nil
	}

	// Marshal message to JSON
	data, err := json.Marshal(output)
	if err != nil {
		return &common.StreamResponse{
			Error: fmt.Errorf("failed to marshal response: %w", err),
			Done:  true,
		}, nil
	}

	return &common.StreamResponse{
		Data:    data,
		Headers: r.extractHeaders(),
	}, nil
}

// Close closes the stream
func (r *GRPCStreamReader) Close() error {
	if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		return common.ErrStreamClosed
	}

	return r.stream.CloseSend()
}

// extractHeaders extracts headers from the gRPC stream
func (r *GRPCStreamReader) extractHeaders() map[string]string {
	headers := make(map[string]string)

	// Extract metadata from stream
	if md, err := r.stream.Header(); err == nil {
		for k, values := range md {
			if len(values) > 0 {
				headers[k] = values[0]
			}
		}
	}

	return headers
}
