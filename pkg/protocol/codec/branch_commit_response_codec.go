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
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
)

func init() {
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchCommitResponseCodec{})
}

type BranchCommitResponseCodec struct {
}

func (g *BranchCommitResponseCodec) Decode(in []byte) interface{} {

	res := message.BranchCommitResponse{}

	buf := binary.NewByteBuf(len(in))
	buf.Write(in)
	var length uint16

	length = ReadUInt16(buf)
	if length > 0 {
		bytes := make([]byte, length)
		res.Xid = string(Read(buf, bytes))
	}
	res.BranchId = int64(ReadUInt64(buf))
	res.BranchStatus = branch.BranchStatus(ReadByte(buf))

	return res
}

func (g *BranchCommitResponseCodec) Encode(in interface{}) []byte {
	req, _ := in.(message.BranchCommitResponse)
	buf := binary.NewByteBuf(0)

	Write16String(req.Xid, buf)
	buf.WriteInt64(req.BranchId)
	buf.WriteByte(byte(req.BranchStatus))

	return buf.RawBuf()
}

func (g *BranchCommitResponseCodec) GetMessageType() message.MessageType {
	return message.MessageType_BranchCommitResult
}
