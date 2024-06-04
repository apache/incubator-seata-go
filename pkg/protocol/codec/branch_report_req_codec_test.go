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

	"github.com/stretchr/testify/assert"

	model2 "seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
)

func TestBranchReportRequestCodec(t *testing.T) {
	msg := message.BranchReportRequest{
		Xid:             "123344",
		ResourceId:      "root:12345678@tcp(127.0.0.1:3306)/seata_client",
		Status:          model2.BranchStatusPhaseoneDone,
		BranchId:        56678,
		BranchType:      model2.BranchTypeAT,
		ApplicationData: []byte("TestExtraData"),
	}

	codec := BranchReportRequestCodec{}
	bytes := codec.Encode(msg)
	msg2 := codec.Decode(bytes)

	assert.Equal(t, msg, msg2)
}
