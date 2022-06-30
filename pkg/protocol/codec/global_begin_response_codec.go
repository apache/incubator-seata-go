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

	"github.com/seata/seata-go/pkg/common/bytes"
	serror "github.com/seata/seata-go/pkg/common/error"
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalBeginResponseCodec{})
}

type GlobalBeginResponseCodec struct {
}

func (c *GlobalBeginResponseCodec) Encode(in interface{}) []byte {
	data := in.(message.GlobalBeginResponse)
	buf := bytes.NewByteBuffer([]byte{})

	buf.WriteByte(byte(data.ResultCode))
	if data.ResultCode == message.ResultCodeFailed {
		var msg string
		if len(data.Msg) > math.MaxInt16 {
			msg = data.Msg[:math.MaxInt16]
		} else {
			msg = data.Msg
		}
		bytes.WriteString16Length(msg, buf)
	}
	buf.WriteByte(byte(data.TransactionExceptionCode))
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
	data.TransactionExceptionCode = serror.TransactionExceptionCode(bytes.ReadByte(buf))
	data.Xid = bytes.ReadString16Length(buf)
	data.ExtraData = []byte(bytes.ReadString16Length(buf))

	return data
}

func (g *GlobalBeginResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalBeginResult
}
