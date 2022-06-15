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
	"github.com/fagongzi/goetty"
)

import (
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/transaction"
)

func init() {
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalBeginResponseCodec{})
}

type GlobalBeginResponseCodec struct {
}

func (c *GlobalBeginResponseCodec) Encode(in interface{}) []byte {
	buf := goetty.NewByteBuf(0)
	resp := in.(message.GlobalBeginResponse)

	buf.WriteByte(byte(resp.ResultCode))
	if resp.ResultCode == message.ResultCodeFailed {
		var msg string
		if len(resp.Msg) > 128 {
			msg = resp.Msg[:128]
		} else {
			msg = resp.Msg
		}
		Write16String(msg, buf)
	}
	buf.WriteByte(byte(resp.TransactionExceptionCode))
	Write16String(resp.Xid, buf)
	Write16String(string(resp.ExtraData), buf)

	return buf.RawBuf()
}

func (g *GlobalBeginResponseCodec) Decode(in []byte) interface{} {
	var lenth uint16
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)
	msg := message.GlobalBeginResponse{}

	resultCode := ReadByte(buf)
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		lenth = ReadUInt16(buf)
		if lenth > 0 {
			bytes := make([]byte, lenth)
			msg.Msg = string(Read(buf, bytes))
		}
	}

	exceptionCode := ReadByte(buf)
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

	lenth = ReadUInt16(buf)
	if lenth > 0 {
		bytes := make([]byte, lenth)
		msg.Xid = string(Read(buf, bytes))
	}

	lenth = ReadUInt16(buf)
	if lenth > 0 {
		bytes := make([]byte, lenth)
		msg.ExtraData = Read(buf, bytes)
	}

	return msg
}

func (g *GlobalBeginResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_GlobalBeginResult
}
