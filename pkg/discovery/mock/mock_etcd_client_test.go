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

package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestNewMockEtcdClient tests the creation of a new MockEtcdClient
func TestNewMockEtcdClient(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Create new mock etcd client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			assert.NotNil(t, mockClient)
			assert.NotNil(t, mockClient.recorder)
		})
	}
}

// TestMockEtcdClient_EXPECT tests the EXPECT() method
func TestMockEtcdClient_EXPECT(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "EXPECT returns recorder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			recorder := mockClient.EXPECT()
			assert.NotNil(t, recorder)
			assert.Equal(t, mockClient.recorder, recorder)
		})
	}
}

// TestMockEtcdClient_Get tests the Get method with mock expectations
func TestMockEtcdClient_Get(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		expectedResp   *clientv3.GetResponse
		expectedErr    error
		setupMock      bool
		validateCalled bool
	}{
		{
			name: "Successful Get operation",
			key:  "test-key",
			expectedResp: &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("test-key"),
						Value: []byte("test-value"),
					},
				},
			},
			expectedErr:    nil,
			setupMock:      true,
			validateCalled: true,
		},
		{
			name:           "Get operation with error",
			key:            "error-key",
			expectedResp:   nil,
			expectedErr:    errors.New("get failed"),
			setupMock:      true,
			validateCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			if tt.setupMock {
				mockClient.EXPECT().
					Get(gomock.Any(), tt.key).
					Return(tt.expectedResp, tt.expectedErr).
					Times(1)
			}

			resp, err := mockClient.Get(ctx, tt.key)

			if tt.validateCalled {
				assert.Equal(t, tt.expectedResp, resp)
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

// TestMockEtcdClient_Put tests the Put method
func TestMockEtcdClient_Put(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		expectedResp *clientv3.PutResponse
		expectedErr  error
	}{
		{
			name:         "Successful Put operation",
			key:          "test-key",
			value:        "test-value",
			expectedResp: &clientv3.PutResponse{},
			expectedErr:  nil,
		},
		{
			name:         "Put operation with error",
			key:          "error-key",
			value:        "error-value",
			expectedResp: nil,
			expectedErr:  errors.New("put failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				Put(gomock.Any(), tt.key, tt.value).
				Return(tt.expectedResp, tt.expectedErr).
				Times(1)

			resp, err := mockClient.Put(ctx, tt.key, tt.value)

			assert.Equal(t, tt.expectedResp, resp)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// TestMockEtcdClient_Delete tests the Delete method
func TestMockEtcdClient_Delete(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		expectedResp *clientv3.DeleteResponse
		expectedErr  error
	}{
		{
			name: "Successful Delete operation",
			key:  "test-key",
			expectedResp: &clientv3.DeleteResponse{
				Deleted: 1,
			},
			expectedErr: nil,
		},
		{
			name:         "Delete operation with error",
			key:          "error-key",
			expectedResp: nil,
			expectedErr:  errors.New("delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				Delete(gomock.Any(), tt.key).
				Return(tt.expectedResp, tt.expectedErr).
				Times(1)

			resp, err := mockClient.Delete(ctx, tt.key)

			assert.Equal(t, tt.expectedResp, resp)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// TestMockEtcdClient_Watch tests the Watch method
func TestMockEtcdClient_Watch(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		expectedChan clientv3.WatchChan
	}{
		{
			name:         "Successful Watch operation",
			key:          "test-key",
			expectedChan: make(clientv3.WatchChan),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				Watch(gomock.Any(), tt.key).
				Return(tt.expectedChan).
				Times(1)

			watchChan := mockClient.Watch(ctx, tt.key)

			assert.Equal(t, tt.expectedChan, watchChan)
		})
	}
}

// TestMockEtcdClient_Close tests the Close method
func TestMockEtcdClient_Close(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "Successful Close operation",
			expectedErr: nil,
		},
		{
			name:        "Close operation with error",
			expectedErr: errors.New("close failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)

			mockClient.EXPECT().
				Close().
				Return(tt.expectedErr).
				Times(1)

			err := mockClient.Close()

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// TestMockEtcdClient_Compact tests the Compact method
func TestMockEtcdClient_Compact(t *testing.T) {
	tests := []struct {
		name         string
		rev          int64
		expectedResp *clientv3.CompactResponse
		expectedErr  error
	}{
		{
			name:         "Successful Compact operation",
			rev:          100,
			expectedResp: &clientv3.CompactResponse{},
			expectedErr:  nil,
		},
		{
			name:         "Compact operation with error",
			rev:          50,
			expectedResp: nil,
			expectedErr:  errors.New("compact failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				Compact(gomock.Any(), tt.rev).
				Return(tt.expectedResp, tt.expectedErr).
				Times(1)

			resp, err := mockClient.Compact(ctx, tt.rev)

			assert.Equal(t, tt.expectedResp, resp)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// TestMockEtcdClient_Do tests the Do method
func TestMockEtcdClient_Do(t *testing.T) {
	tests := []struct {
		name         string
		op           clientv3.Op
		expectedResp clientv3.OpResponse
		expectedErr  error
	}{
		{
			name:         "Successful Do operation",
			op:           clientv3.OpGet("test-key"),
			expectedResp: clientv3.OpResponse{},
			expectedErr:  nil,
		},
		{
			name:         "Do operation with error",
			op:           clientv3.OpGet("error-key"),
			expectedResp: clientv3.OpResponse{},
			expectedErr:  errors.New("do failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				Return(tt.expectedResp, tt.expectedErr).
				Times(1)

			resp, err := mockClient.Do(ctx, tt.op)

			assert.Equal(t, tt.expectedResp, resp)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// TestMockEtcdClient_Txn tests the Txn method
func TestMockEtcdClient_Txn(t *testing.T) {
	tests := []struct {
		name        string
		expectedTxn clientv3.Txn
	}{
		{
			name:        "Successful Txn operation",
			expectedTxn: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				Txn(gomock.Any()).
				Return(tt.expectedTxn).
				Times(1)

			txn := mockClient.Txn(ctx)

			assert.Equal(t, tt.expectedTxn, txn)
		})
	}
}

// TestMockEtcdClient_RequestProgress tests the RequestProgress method
func TestMockEtcdClient_RequestProgress(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "Successful RequestProgress operation",
			expectedErr: nil,
		},
		{
			name:        "RequestProgress operation with error",
			expectedErr: errors.New("request progress failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				RequestProgress(gomock.Any()).
				Return(tt.expectedErr).
				Times(1)

			err := mockClient.RequestProgress(ctx)

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// TestMockEtcdClient_MultipleMethodCalls tests multiple method calls in sequence
func TestMockEtcdClient_MultipleMethodCalls(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Multiple operations in sequence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			// Setup expectations for multiple operations
			mockClient.EXPECT().
				Put(gomock.Any(), "key1", "value1").
				Return(&clientv3.PutResponse{}, nil).
				Times(1)

			mockClient.EXPECT().
				Get(gomock.Any(), "key1").
				Return(&clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte("key1"), Value: []byte("value1")},
					},
				}, nil).
				Times(1)

			mockClient.EXPECT().
				Delete(gomock.Any(), "key1").
				Return(&clientv3.DeleteResponse{Deleted: 1}, nil).
				Times(1)

			// Execute operations
			_, err := mockClient.Put(ctx, "key1", "value1")
			assert.NoError(t, err)

			resp, err := mockClient.Get(ctx, "key1")
			assert.NoError(t, err)
			assert.Len(t, resp.Kvs, 1)

			delResp, err := mockClient.Delete(ctx, "key1")
			assert.NoError(t, err)
			assert.Equal(t, int64(1), delResp.Deleted)
		})
	}
}

// TestMockEtcdClient_GetWithOptions tests Get with options
func TestMockEtcdClient_GetWithOptions(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		opts         []clientv3.OpOption
		expectedResp *clientv3.GetResponse
		expectedErr  error
	}{
		{
			name: "Get with WithPrefix option",
			key:  "test-",
			opts: []clientv3.OpOption{clientv3.WithPrefix()},
			expectedResp: &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{Key: []byte("test-key1"), Value: []byte("value1")},
					{Key: []byte("test-key2"), Value: []byte("value2")},
				},
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			mockClient.EXPECT().
				Get(gomock.Any(), tt.key, gomock.Any()).
				Return(tt.expectedResp, tt.expectedErr).
				Times(1)

			resp, err := mockClient.Get(ctx, tt.key, tt.opts...)

			assert.Equal(t, tt.expectedResp, resp)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// TestMockEtcdClient_ControllerIntegration tests mock controller integration
func TestMockEtcdClient_ControllerIntegration(t *testing.T) {
	tests := []struct {
		name        string
		shouldPanic bool
	}{
		{
			name:        "Controller verifies all expectations met",
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockClient := NewMockEtcdClient(ctrl)
			ctx := context.Background()

			// Setup expectations
			mockClient.EXPECT().
				Get(gomock.Any(), "test-key").
				Return(&clientv3.GetResponse{}, nil).
				Times(1)

			// Call the method
			_, err := mockClient.Get(ctx, "test-key")
			assert.NoError(t, err)

			// Finish should not panic if all expectations are met
			if !tt.shouldPanic {
				ctrl.Finish()
			}
		})
	}
}
