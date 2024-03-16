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
	"time"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/bytes"
)

type GlobalBeginRequestCodec struct{}

func (c *GlobalBeginRequestCodec) Encode(in interface{}) []byte {
	data := in.(message.GlobalBeginRequest)
	buf := bytes.NewByteBuffer([]byte{})
	re := uint32(int64(data.Timeout) / 1e6)
	buf.WriteUint32(re)
	bytes.WriteString16Length(data.TransactionName, buf)

	return buf.Bytes()
}

func (g *GlobalBeginRequestCodec) Decode(in []byte) interface{} {
	data := message.GlobalBeginRequest{}
	buf := bytes.NewByteBuffer(in)
	re := int64(bytes.ReadUInt32(buf)) * 1e6
	data.Timeout = time.Duration(re)
	data.TransactionName = bytes.ReadString16Length(buf)

	return data
}

func (g *GlobalBeginRequestCodec) GetMessageType() message.MessageType {
	return message.MessageTypeGlobalBegin
}
