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
	model2 "seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/bytes"
)

type GlobalLockQueryRequestCodec struct {
}

func (g *GlobalLockQueryRequestCodec) Decode(in []byte) interface{} {
	data := message.GlobalLockQueryRequest{}
	buf := bytes.NewByteBuffer(in)

	data.Xid = bytes.ReadString16Length(buf)
	data.BranchType = model2.BranchType(bytes.ReadByte(buf))
	data.ResourceId = bytes.ReadString16Length(buf)
	data.LockKey = bytes.ReadString32Length(buf)
	data.ApplicationData = []byte(bytes.ReadString32Length(buf))

	return data
}

func (c *GlobalLockQueryRequestCodec) Encode(in interface{}) []byte {
	data, _ := in.(message.GlobalLockQueryRequest)
	buf := bytes.NewByteBuffer([]byte{})

	bytes.WriteString16Length(data.Xid, buf)
	buf.WriteByte(byte(data.BranchType))
	bytes.WriteString16Length(data.ResourceId, buf)
	bytes.WriteString32Length(data.LockKey, buf)
	bytes.WriteString32Length(string(data.ApplicationData), buf)

	return buf.Bytes()
}

func (g *GlobalLockQueryRequestCodec) GetMessageType() message.MessageType {
	return message.MessageTypeGlobalLockQuery
}
