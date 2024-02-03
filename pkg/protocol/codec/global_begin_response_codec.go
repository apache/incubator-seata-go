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

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/bytes"
	serror "seata.apache.org/seata-go/pkg/util/errors"
)

type GlobalBeginResponseCodec struct{}

func (c *GlobalBeginResponseCodec) Encode(in interface{}) []byte {
	data := in.(message.GlobalBeginResponse)
	buf := bytes.NewByteBuffer([]byte{})

	buf.WriteByte(byte(data.ResultCode))
	if data.ResultCode == message.ResultCodeFailed {
		msg := data.Msg
		if len(data.Msg) > math.MaxInt16 {
			msg = data.Msg[:math.MaxInt16]
		}
		bytes.WriteString16Length(msg, buf)
	}
	buf.WriteByte(byte(data.TransactionErrorCode))
	bytes.WriteString16Length(data.Xid, buf)
	bytes.WriteString16Length(string(data.ExtraData), buf)

	return buf.Bytes()
}

func (g *GlobalBeginResponseCodec) Decode(in []byte) interface{} {
	data := message.GlobalBeginResponse{}
	buf := bytes.NewByteBuffer(in)

	data.ResultCode = message.ResultCode(bytes.ReadByte(buf))
	if data.ResultCode == message.ResultCodeFailed {
		data.Msg = bytes.ReadString16Length(buf)
	}
	data.TransactionErrorCode = serror.TransactionErrorCode(bytes.ReadByte(buf))
	data.Xid = bytes.ReadString16Length(buf)
	data.ExtraData = []byte(bytes.ReadString16Length(buf))

	return data
}

func (g *GlobalBeginResponseCodec) GetMessageType() message.MessageType {
	return message.MessageTypeGlobalBeginResult
}
