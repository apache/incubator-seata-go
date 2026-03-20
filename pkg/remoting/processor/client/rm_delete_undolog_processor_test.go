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
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
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
				Type: message.GettyRequestTypeRequestSync,
				Body: message.UndoLogDeleteRequest{
					ResourceId: "jdbc:mysql://localhost:3306/seata",
					SaveDays:   7,
					BranchType: branch.BranchTypeAT,
				},
			},
			wantErr: false,
		},
		{
			name: "process undo log delete request with empty resource id",
			rpcMessage: message.RpcMessage{
				ID:   2,
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
				ID:   3,
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
				ID:   4,
				Type: message.GettyRequestTypeRequestOneway,
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
			err := processor.Process(context.Background(), tt.rpcMessage)
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

	t.Run("background context", func(t *testing.T) {
		err := processor.Process(context.Background(), message.RpcMessage{
			ID:   1,
			Type: message.GettyRequestTypeRequestSync,
			Body: message.UndoLogDeleteRequest{
				ResourceId: "test-resource-1",
				SaveDays:   7,
				BranchType: branch.BranchTypeAT,
			},
		})
		assert.NoError(t, err)
	})

	t.Run("cancelled context should be handled gracefully", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		
		err := processor.Process(ctx, message.RpcMessage{
			ID:   2,
			Type: message.GettyRequestTypeRequestSync,
			Body: message.UndoLogDeleteRequest{
				ResourceId: "test-resource-2",
				SaveDays:   7,
				BranchType: branch.BranchTypeAT,
			},
		})
		assert.NoError(t, err, "current impl does not use ctx; update when deletion logic is added")
	})
}

func TestInitDeleteUndoLog(t *testing.T) {
	getty.GetGettyClientHandlerInstance()

	assert.NotPanics(t, func() {
		initDeleteUndoLog()
	}, "initDeleteUndoLog should not panic")
}

func TestRmDeleteUndoLogProcessor_Integration(t *testing.T) {
	RegisterProcessor()

	processor := &rmDeleteUndoLogProcessor{}

	err := processor.Process(context.Background(), message.RpcMessage{
		ID:    100,
		Type:  message.GettyRequestTypeRequestSync,
		Codec: 1,
		Body: message.UndoLogDeleteRequest{
			ResourceId: "jdbc:mysql://127.0.0.1:3306/seata_test",
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	})
	assert.NoError(t, err)
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
					SaveDays:   32767,
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
					SaveDays:   -32768,
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
			err := processor.Process(context.Background(), tt.rpcMessage)
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
