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

package codec

import (
	"testing"

	model2 "github.com/seata/seata-go/pkg/protocol/branch"

	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/protocol/message"
)

func TestBranchRollbackRequestCodec(t *testing.T) {
	msg := message.BranchRollbackRequest{
		AbstractBranchEndRequest: message.AbstractBranchEndRequest{
			Xid:             "123344",
			BranchId:        56678,
			BranchType:      model2.BranchTypeSAGA,
			ResourceId:      "1232323",
			ApplicationData: []byte("TestExtraData"),
		},
	}

	codec := BranchRollbackRequestCodec{}
	bytes := codec.Encode(msg)
	msg2 := codec.Decode(bytes)

	assert.Equal(t, msg, msg2)
}
