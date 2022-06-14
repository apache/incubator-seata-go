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
)

type AbstractIdentifyRequestCodec struct {
}

func (c *AbstractIdentifyRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.AbstractIdentifyRequest)
	buf := goetty.NewByteBuf(0)

	Write16String(req.Version, buf)
	Write16String(req.ApplicationId, buf)
	Write16String(req.TransactionServiceGroup, buf)
	Write16String(string(req.ExtraData), buf)

	return buf.RawBuf()
}

func (c *AbstractIdentifyRequestCodec) Decode(in []byte) interface{} {
	msg := message.AbstractIdentifyRequest{}
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)
	var len uint16

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if uint16(buf.Readable()) < len {
		return msg
	}
	versionBytes := make([]byte, len)
	msg.Version = string(Read(buf, versionBytes))

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if uint16(buf.Readable()) < len {
		return msg
	}
	applicationIdBytes := make([]byte, len)
	msg.ApplicationId = string(Read(buf, applicationIdBytes))

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if uint16(buf.Readable()) < len {
		return msg
	}
	transactionServiceGroupBytes := make([]byte, len)
	msg.TransactionServiceGroup = string(Read(buf, transactionServiceGroupBytes))

	if buf.Readable() < 2 {
		return msg
	}
	len = ReadUInt16(buf)
	if len > 0 && uint16(buf.Readable()) > len {
		extraDataBytes := make([]byte, len)
		msg.ExtraData = Read(buf, extraDataBytes)
	}

	return msg
}
