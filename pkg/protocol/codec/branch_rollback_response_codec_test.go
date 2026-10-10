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
	"math"
	"strings"
	"testing"

	serror "seata.apache.org/seata-go/v2/pkg/util/errors"

	model2 "seata.apache.org/seata-go/v2/pkg/protocol/branch"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
)

func TestBranchRollbackResponseCodec(t *testing.T) {
	msg := message.BranchRollbackResponse{
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

	codec := BranchRollbackResponseCodec{}
	bytes := codec.Encode(msg)
	msg2 := codec.Decode(bytes)

	assert.Equal(t, msg, msg2)
}

func TestBranchRollbackResponseCodec_TruncatesLongMessageWithoutShiftingFields(t *testing.T) {
	msg := message.BranchRollbackResponse{
		AbstractBranchEndResponse: message.AbstractBranchEndResponse{
			Xid:          "xid",
			BranchId:     123,
			BranchStatus: model2.BranchStatusPhasetwoRollbackFailedRetryable,
			AbstractTransactionResponse: message.AbstractTransactionResponse{
				TransactionErrorCode: serror.TransactionErrorCodeBranchRollbackFailedRetriable,
				AbstractResultMessage: message.AbstractResultMessage{
					ResultCode: message.ResultCodeFailed,
					Msg:        strings.Repeat("x", math.MaxInt16+100),
				},
			},
		},
	}

	codec := BranchRollbackResponseCodec{}
	decoded := codec.Decode(codec.Encode(msg)).(message.BranchRollbackResponse)

	assert.Equal(t, message.ResultCodeFailed, decoded.ResultCode)
	assert.Equal(t, strings.Repeat("x", math.MaxInt16), decoded.Msg)
	assert.Equal(t, msg.TransactionErrorCode, decoded.TransactionErrorCode)
	assert.Equal(t, msg.Xid, decoded.Xid)
	assert.Equal(t, msg.BranchId, decoded.BranchId)
	assert.Equal(t, msg.BranchStatus, decoded.BranchStatus)
}

// Byte vector derived from Java AbstractResultMessageCodec,
// AbstractTransactionResponseCodec, and AbstractBranchEndResponseCodec.
func TestBranchRollbackResponseCodec_JavaWireFormat(t *testing.T) {
	msg := message.BranchRollbackResponse{
		AbstractBranchEndResponse: message.AbstractBranchEndResponse{
			Xid:          "192.168.0.1:8091:1234",
			BranchId:     5678,
			BranchStatus: model2.BranchStatusPhasetwoRollbackFailedRetryable,
			AbstractTransactionResponse: message.AbstractTransactionResponse{
				TransactionErrorCode: serror.TransactionErrorCodeBranchRollbackFailedRetriable,
				AbstractResultMessage: message.AbstractResultMessage{
					ResultCode: message.ResultCodeFailed,
					Msg:        "storage failed",
				},
			},
		},
	}
	want := []byte{
		0x00,
		0x00, 0x0e,
		's', 't', 'o', 'r', 'a', 'g', 'e', ' ', 'f', 'a', 'i', 'l', 'e', 'd',
		0x04,
		0x00, 0x15,
		'1', '9', '2', '.', '1', '6', '8', '.', '0', '.', '1', ':', '8', '0', '9', '1', ':', '1', '2', '3', '4',
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x16, 0x2e,
		0x09,
	}

	codec := BranchRollbackResponseCodec{}
	assert.Equal(t, want, codec.Encode(msg), "encode must reproduce the Java bytes exactly")
	assert.Equal(t, msg, codec.Decode(want))
}
