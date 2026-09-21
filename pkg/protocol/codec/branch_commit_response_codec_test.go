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
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	model2 "seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	serror "seata.apache.org/seata-go/v2/pkg/util/errors"
)

func TestBranchCommitResponseCodec(t *testing.T) {
	msg := message.BranchCommitResponse{
		AbstractBranchEndResponse: message.AbstractBranchEndResponse{
			Xid:          "123344",
			BranchId:     56678,
			BranchStatus: model2.BranchStatusPhaseoneFailed,
			AbstractTransactionResponse: message.AbstractTransactionResponse{
				TransactionErrorCode: serror.TransactionErrorCodeBeginFailed,
				AbstractResultMessage: message.AbstractResultMessage{
					ResultCode: message.ResultCodeFailed,
					Msg:        "FAILED",
				},
			},
		},
	}

	codec := BranchCommitResponseCodec{}
	encoded := codec.Encode(msg)
	assert.Equal(t, uint16(len(msg.Msg)), binary.BigEndian.Uint16(encoded[1:3]))
	assert.Equal(t, msg.Msg, string(encoded[3:3+len(msg.Msg)]))
	msg2 := codec.Decode(encoded)

	assert.Equal(t, msg, msg2)
}

func TestBranchCommitResponseCodecTruncatesLongMessage(t *testing.T) {
	msg := message.BranchCommitResponse{
		AbstractBranchEndResponse: message.AbstractBranchEndResponse{
			AbstractTransactionResponse: message.AbstractTransactionResponse{
				AbstractResultMessage: message.AbstractResultMessage{
					ResultCode: message.ResultCodeFailed,
					Msg:        strings.Repeat("x", math.MaxInt16+1),
				},
			},
		},
	}

	codec := BranchCommitResponseCodec{}
	encoded := codec.Encode(msg)
	assert.Equal(t, uint16(math.MaxInt16), binary.BigEndian.Uint16(encoded[1:3]))
	decoded := codec.Decode(encoded).(message.BranchCommitResponse)
	assert.Len(t, decoded.Msg, math.MaxInt16)
}
