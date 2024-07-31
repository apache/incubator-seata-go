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
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/bytes"
)

type RegisterRMRequestCodec struct{}

func (g *RegisterRMRequestCodec) Decode(in []byte) interface{} {
	data := message.RegisterRMRequest{}
	buf := bytes.NewByteBuffer(in)

	data.Version = bytes.ReadString16Length(buf)
	data.ApplicationId = bytes.ReadString16Length(buf)
	data.TransactionServiceGroup = bytes.ReadString16Length(buf)
	data.ExtraData = []byte(bytes.ReadString16Length(buf))
	data.ResourceIds = bytes.ReadString32Length(buf)

	return data
}

func (c *RegisterRMRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.RegisterRMRequest)
	buf := bytes.NewByteBuffer([]byte{})

	bytes.WriteString16Length(req.Version, buf)
	bytes.WriteString16Length(req.ApplicationId, buf)
	bytes.WriteString16Length(req.TransactionServiceGroup, buf)
	bytes.WriteString16Length(string(req.ExtraData), buf)
	bytes.WriteString32Length(req.ResourceIds, buf)

	return buf.Bytes()
}

func (g *RegisterRMRequestCodec) GetMessageType() message.MessageType {
	return message.MessageTypeRegRm
}
