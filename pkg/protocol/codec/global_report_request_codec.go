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

type GlobalReportRequestCodec struct {
	CommonGlobalEndRequestCodec
}

// Decode decode global report request
func (g *GlobalReportRequestCodec) Decode(in []byte) interface{} {
	data := message.AbstractGlobalEndRequest{}
	buf := bytes.NewByteBuffer(in)

	data.Xid = bytes.ReadString16Length(buf)
	data.ExtraData = []byte(bytes.ReadString16Length(buf))
	status := bytes.ReadByte(buf)

	return message.GlobalReportRequest{
		AbstractGlobalEndRequest: data,
		GlobalStatus:             message.GlobalStatus(status),
	}
}

// Encode encode global report request
func (g *GlobalReportRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.GlobalReportRequest)
	b := g.CommonGlobalEndRequestCodec.Encode(req.AbstractGlobalEndRequest)
	buf := bytes.NewByteBuffer(b)
	buf.WriteByte(byte(req.GlobalStatus))
	return buf.Bytes()
}

// GetMessageType get global report request's message type
func (g *GlobalReportRequestCodec) GetMessageType() message.MessageType {
	return message.MessageTypeGlobalReportResult
}
