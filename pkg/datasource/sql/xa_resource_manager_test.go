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

package sql

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func TestXAResourceManager_LockQuery(t *testing.T) {
	tests := []struct {
		name    string
		resp    interface{}
		respErr error
		want    bool
		wantErr string
	}{
		{
			name: "lockable",
			resp: message.GlobalLockQueryResponse{Lockable: true},
			want: true,
		},
		{
			name: "unlockable",
			resp: message.GlobalLockQueryResponse{Lockable: false},
			want: false,
		},
		{
			name:    "remoting error",
			respErr: errors.New("network timeout"),
			want:    false,
			wantErr: "network timeout",
		},
	}

	param := rm.LockQueryParam{
		BranchType: branch.BranchTypeXA,
		ResourceId: "jdbc:mysql://test/resource",
		Xid:        "test-xid",
		LockKeys:   "user:1",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(getty.GetGettyRemotingClient()), "SendSyncRequest",
				func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
					req, ok := msg.(message.GlobalLockQueryRequest)
					if assert.True(t, ok) {
						assert.Equal(t, param.BranchType, req.BranchType)
						assert.Equal(t, param.ResourceId, req.ResourceId)
						assert.Equal(t, param.Xid, req.Xid)
						assert.Equal(t, param.LockKeys, req.LockKey)
					}
					return tt.resp, tt.respErr
				})
			defer patches.Reset()

			xaManager := &XAResourceManager{rmRemoting: rm.GetRMRemotingInstance()}

			got, err := xaManager.LockQuery(context.Background(), param)

			assert.Equal(t, tt.want, got)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
