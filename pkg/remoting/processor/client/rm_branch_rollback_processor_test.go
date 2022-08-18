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

	model2 "github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/codec"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/rm/tcc"
)

func TestRmBranchRollbackProcessor(t *testing.T) {

	// testcases
	var tests = []struct {
		name    string             // testcase name
		rpcMsg  message.RpcMessage // rpcMessage case
		wantErr bool               //want testcase err or not
	}{
		{
			name: "rbr-testcase1-failure",
			rpcMsg: message.RpcMessage{
				ID:         223,
				Type:       message.GettyRequestType(message.MessageType_BranchRollback),
				Codec:      byte(codec.CodecTypeSeata),
				Compressor: byte(1),
				HeadMap: map[string]string{
					"name":    " Jack",
					"age":     "12",
					"address": "Beijing",
				},
				Body: message.BranchRollbackRequest{
					AbstractBranchEndRequest: message.AbstractBranchEndRequest{
						Xid:             "123345",
						BranchId:        56679,
						BranchType:      model2.BranchTypeTCC,
						ResourceId:      "1232324",
						ApplicationData: []byte("TestExtraData"),
					},
				},
			},

			wantErr: true, // need dail to server, so err accured
		},
	}

	var ctx context.Context
	var rbrProcessor rmBranchRollbackProcessor

	rm.GetRmCacheInstance().RegisterResourceManager(tcc.GetTCCResourceManagerInstance())

	// run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := rbrProcessor.Process(ctx, tc.rpcMsg)
			if (err != nil) != tc.wantErr {
				t.Errorf("rmBranchRollbackProcessor wantErr: %v, got: %v", tc.wantErr, err)
				return
			}
		})
	}

}
