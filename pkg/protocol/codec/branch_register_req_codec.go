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
	"github.com/seata/seata-go/pkg/common/binary"

	model2 "github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchRegisterRequestCodec{})
}

type BranchRegisterRequestCodec struct {
}

func (g *BranchRegisterRequestCodec) Decode(in []byte) interface{} {
	buf := binary.NewByteBuf(len(in))
	buf.Write(in)
	msg := message.BranchRegisterRequest{}

	length := ReadUInt16(buf)
	if length > 0 {
		bytes := make([]byte, length)
		msg.Xid = string(Read(buf, bytes))
	}

	msg.BranchType = model2.BranchType(ReadByte(buf))

	length = ReadUInt16(buf)
	if length > 0 {
		bytes := make([]byte, length)
		msg.ResourceId = string(Read(buf, bytes))
	}

	length32 := ReadUInt32(buf)
	if length > 0 {
		bytes := make([]byte, length32)
		msg.LockKey = string(Read(buf, bytes))
	}

	length32 = ReadUInt32(buf)
	if length > 0 {
		bytes := make([]byte, length32)
		msg.ApplicationData = Read(buf, bytes)
	}

	return msg
}

func (c *BranchRegisterRequestCodec) Encode(in interface{}) []byte {
	buf := binary.NewByteBuf(0)
	req, _ := in.(message.BranchRegisterRequest)

	Write16String(req.Xid, buf)
	buf.WriteByte(byte(req.BranchType))
	Write16String(req.ResourceId, buf)
	Write32String(req.LockKey, buf)
	Write32String(string(req.ApplicationData), buf)

	return buf.RawBuf()
}

func (g *BranchRegisterRequestCodec) GetMessageType() message.MessageType {
	return message.MessageType_BranchRegister
}
