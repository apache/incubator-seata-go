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

	t.Run("Put", func(t *testing.T) {
		expected := &clientv3.PutResponse{}
		m.EXPECT().Put(ctx, "key", "val").Return(expected, nil)
		res, err := m.Put(ctx, "key", "val")
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("Get", func(t *testing.T) {
		expected := &clientv3.GetResponse{}
		m.EXPECT().Get(ctx, "key").Return(expected, nil)
		res, err := m.Get(ctx, "key")
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("Delete", func(t *testing.T) {
		expected := &clientv3.DeleteResponse{}
		m.EXPECT().Delete(ctx, "key").Return(expected, nil)
		res, err := m.Delete(ctx, "key")
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("Compact", func(t *testing.T) {
		expected := &clientv3.CompactResponse{}
		m.EXPECT().Compact(ctx, int64(10)).Return(expected, nil)
		res, err := m.Compact(ctx, 10)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("Do", func(t *testing.T) {
		op := clientv3.Op{}
		resp := clientv3.OpResponse{}
		m.EXPECT().Do(ctx, op).Return(resp, nil)
		r, err := m.Do(ctx, op)
		assert.NoError(t, err)
		assert.Equal(t, resp, r)
	})

	t.Run("Txn", func(t *testing.T) {
		m.EXPECT().Txn(ctx).Return(nil)
		res := m.Txn(ctx)
		assert.Nil(t, res)
	})

	t.Run("RequestProgress", func(t *testing.T) {
		m.EXPECT().RequestProgress(ctx).Return(nil)
		err := m.RequestProgress(ctx)
		assert.NoError(t, err)
	})

	t.Run("Watch", func(t *testing.T) {
		watchChan := make(clientv3.WatchChan)
		m.EXPECT().Watch(ctx, "key").Return(watchChan)
		res := m.Watch(ctx, "key")
		assert.Equal(t, watchChan, res)
	})

	t.Run("Close", func(t *testing.T) {
		m.EXPECT().Close().Return(errors.New("close error"))
		err := m.Close()
		assert.EqualError(t, err, "close error")
	})
}
