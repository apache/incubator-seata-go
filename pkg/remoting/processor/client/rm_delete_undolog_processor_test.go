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

package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
)

func TestRmDeleteUndoLogProcessor_Process(t *testing.T) {
	processor := &rmDeleteUndoLogProcessor{}

	tests := []struct {
		name       string
		rpcMessage message.RpcMessage
		wantErr    bool
	}{
		{
			name: "process normal undo log delete request",
			rpcMessage: message.RpcMessage{
				ID:   1,
				Type: message.GettyRequestTypeRequestSync, // 修正：使用正确的常量
				Body: message.UndoLogDeleteRequest{
					ResourceId: "jdbc:mysql://localhost:3306/seata",
					SaveDays:   7,
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
		{
			name: "process undo log delete request with TCC branch type",
			rpcMessage: message.RpcMessage{
				ID:   2,
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "tcc-resource",
					SaveDays:   15,
					BranchType: branch.BranchTypeTCC,
				},
			},
			wantErr: false,
		},
		{
			name: "process undo log delete request with XA branch type",
			rpcMessage: message.RpcMessage{
				ID:   3,
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "xa-resource",
					SaveDays:   30,
					BranchType: branch.BranchTypeXA,
				},
			},
			wantErr: false,
		},
		{
			name: "process undo log delete request with empty resource id",
			rpcMessage: message.RpcMessage{
				ID:   4,
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "",
					SaveDays:   10,
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
		{
			name: "process undo log delete request with zero save days",
			rpcMessage: message.RpcMessage{
				ID:   5,
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "test-resource",
					SaveDays:   0,
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
		{
			name: "process with oneway request type",
			rpcMessage: message.RpcMessage{
				ID:   6,
				Type: message.GettyRequestTypeRequestOneway, // TC 服务器可能使用单向请求
				Body: message.UndoLogDeleteRequest{
					ResourceId: "test-resource",
					SaveDays:   7,
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := processor.Process(ctx, tt.rpcMessage)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRmDeleteUndoLogProcessor_ProcessWithContext(t *testing.T) {
	processor := &rmDeleteUndoLogProcessor{}

	tests := []struct {
		name    string
		ctx     context.Context
		req     message.UndoLogDeleteRequest
		wantErr bool
	}{
		{
			name: "process with background context",
			ctx:  context.Background(),
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource-1",
				SaveDays:   7,
				BranchType: branch.BranchTypeAT,
			},
			wantErr: false,
		},
		{
			name: "process with custom context",
			ctx:  context.WithValue(context.Background(), "test-key", "test-value"),
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource-2",
				SaveDays:   14,
				BranchType: branch.BranchTypeTCC,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcMsg := message.RpcMessage{
				ID:   1,
				Type: message.GettyRequestTypeRequestSync,
				Body: tt.req,
			}

			err := processor.Process(tt.ctx, rpcMsg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitDeleteUndoLog(t *testing.T) {
	// Initialize getty client handler (this is normally done at startup)
	getty.GetGettyClientHandlerInstance()

	// Call the init function
	initDeleteUndoLog()

	// Verify processor is registered
	// Note: This test verifies that initDeleteUndoLog doesn't panic
	// The actual registration verification would require accessing internal state
	// of getty.GetGettyClientHandlerInstance(), which may not be exposed

	assert.NotPanics(t, func() {
		initDeleteUndoLog()
	}, "initDeleteUndoLog should not panic")
}

func TestRmDeleteUndoLogProcessor_Integration(t *testing.T) {
	// This test verifies the processor can be registered and works end-to-end
	RegisterProcessor() // Register all processors including delete undo log

	processor := &rmDeleteUndoLogProcessor{}
	ctx := context.Background()

	// Create a realistic RPC message
	rpcMsg := message.RpcMessage{
		ID:    100,
		Type:  message.GettyRequestTypeRequestSync,
		Codec: 1,
		Body: message.UndoLogDeleteRequest{
			ResourceId: "jdbc:mysql://127.0.0.1:3306/seata_test",
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	}

	err := processor.Process(ctx, rpcMsg)
	assert.NoError(t, err, "processing should succeed")
}

func TestRmDeleteUndoLogProcessor_EdgeCases(t *testing.T) {
	processor := &rmDeleteUndoLogProcessor{}

	tests := []struct {
		name       string
		rpcMessage message.RpcMessage
		wantErr    bool
	}{
		{
			name: "max int16 save days",
			rpcMessage: message.RpcMessage{
				ID:   1,
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "test-resource",
					SaveDays:   32767, // max int16
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
		{
			name: "min int16 save days",
			rpcMessage: message.RpcMessage{
				ID:   2,
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "test-resource",
					SaveDays:   -32768, // min int16
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
		{
			name: "very long resource id",
			rpcMessage: message.RpcMessage{
				ID:   3,
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "jdbc:mysql://very-long-hostname-that-could-be-used-in-production.example.com:3306/database_name_with_underscores",
					SaveDays:   7,
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := processor.Process(ctx, tt.rpcMessage)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkRmDeleteUndoLogProcessor_Process(b *testing.B) {
	processor := &rmDeleteUndoLogProcessor{}
	ctx := context.Background()
	rpcMsg := message.RpcMessage{
		ID:   1,
		Type: message.GettyRequestTypeRequestSync,
		Body: message.UndoLogDeleteRequest{
			ResourceId: "jdbc:mysql://localhost:3306/seata",
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.Process(ctx, rpcMsg)
	}
}
