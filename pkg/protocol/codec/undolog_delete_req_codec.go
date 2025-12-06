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
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/bytes"
)

type UndoLogDeleteRequestCodec struct{}

func (u *UndoLogDeleteRequestCodec) Decode(in []byte) interface{} {
	data := message.UndoLogDeleteRequest{}
	buf := bytes.NewByteBuffer(in)

	data.ResourceId = bytes.ReadString16Length(buf)
	saveDays, _ := buf.ReadUint16()
	data.SaveDays = int16(saveDays)
	data.BranchType = branch.BranchType(bytes.ReadByte(buf))

	return data
}

func (u *UndoLogDeleteRequestCodec) Encode(in interface{}) []byte {
	data, _ := in.(message.UndoLogDeleteRequest)
	buf := bytes.NewByteBuffer([]byte{})

	bytes.WriteString16Length(data.ResourceId, buf)
	buf.WriteUint16(uint16(data.SaveDays))
	buf.WriteByte(byte(data.BranchType))

	return buf.Bytes()
}

func (u *UndoLogDeleteRequestCodec) GetMessageType() message.MessageType {
	return message.MessageTypeRmDeleteUndolog
}
