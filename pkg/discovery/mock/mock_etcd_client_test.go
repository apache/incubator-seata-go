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
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestNewMockEtcdClient_BasicUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockEtcdClient(ctrl)
	assert.NotNil(t, mockClient)
	assert.NotNil(t, mockClient.EXPECT())
}

func TestMockEtcdClient_Methods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockEtcdClient(ctrl)
	ctx := context.Background()

	tests := []struct {
		name     string
		mockFunc func()
		run      func(t *testing.T)
	}{
		{
			name: "Put",
			mockFunc: func() {
				expected := &clientv3.PutResponse{}
				m.EXPECT().Put(ctx, "key", "val").Return(expected, nil)
			},
			run: func(t *testing.T) {
				res, err := m.Put(ctx, "key", "val")
				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
		},
		{
			name: "Get",
			mockFunc: func() {
				expected := &clientv3.GetResponse{}
				m.EXPECT().Get(ctx, "key").Return(expected, nil)
			},
			run: func(t *testing.T) {
				res, err := m.Get(ctx, "key")
				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
		},
		{
			name: "Delete",
			mockFunc: func() {
				expected := &clientv3.DeleteResponse{}
				m.EXPECT().Delete(ctx, "key").Return(expected, nil)
			},
			run: func(t *testing.T) {
				res, err := m.Delete(ctx, "key")
				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
		},
		{
			name: "Compact",
			mockFunc: func() {
				expected := &clientv3.CompactResponse{}
				m.EXPECT().Compact(ctx, int64(10)).Return(expected, nil)
			},
			run: func(t *testing.T) {
				res, err := m.Compact(ctx, 10)
				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
		},
		{
			name: "Do",
			mockFunc: func() {
				op := clientv3.Op{}
				resp := clientv3.OpResponse{}
				m.EXPECT().Do(ctx, op).Return(resp, nil)
			},
			run: func(t *testing.T) {
				r, err := m.Do(ctx, clientv3.Op{})
				assert.NoError(t, err)
				assert.NotNil(t, r)
			},
		},
		{
			name: "Txn",
			mockFunc: func() {
				m.EXPECT().Txn(ctx).Return(nil)
			},
			run: func(t *testing.T) {
				res := m.Txn(ctx)
				assert.Nil(t, res)
			},
		},
		{
			name: "RequestProgress",
			mockFunc: func() {
				m.EXPECT().RequestProgress(ctx).Return(nil)
			},
			run: func(t *testing.T) {
				err := m.RequestProgress(ctx)
				assert.NoError(t, err)
			},
		},
		{
			name: "Watch",
			mockFunc: func() {
				watchChan := make(clientv3.WatchChan)
				m.EXPECT().Watch(ctx, "key").Return(watchChan)
			},
			run: func(t *testing.T) {
				res := m.Watch(ctx, "key")
				assert.NotNil(t, res)
			},
		},
		{
			name: "Close",
			mockFunc: func() {
				m.EXPECT().Close().Return(errors.New("close error"))
			},
			run: func(t *testing.T) {
				err := m.Close()
				assert.EqualError(t, err, "close error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			tt.run(t)
		})
	}
}
