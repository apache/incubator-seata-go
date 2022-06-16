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
	"github.com/seata/seata-go/pkg/protocol/message"
)

type AbstractIdentifyResponseCodec struct {
}

func (c *AbstractIdentifyResponseCodec) Encode(in interface{}) []byte {
	buf := goetty.NewByteBuf(0)
	resp := in.(message.AbstractIdentifyResponse)

	if resp.Identified {
		buf.WriteByte(byte(1))
	} else {
		buf.WriteByte(byte(0))
	}

	Write16String(resp.Version, buf)
	return buf.RawBuf()
}

func (c *AbstractIdentifyResponseCodec) Decode(in []byte) interface{} {
	buf := goetty.NewByteBuf(len(in))
	buf.Write(in)
	msg := message.AbstractIdentifyResponse{}

	identified, _ := buf.ReadByte()
	if identified == byte(1) {
		msg.Identified = true
	} else if identified == byte(0) {
		msg.Identified = false
	}

	length := ReadUInt16(buf)
	if length > 0 {
		versionBytes := make([]byte, length)
		msg.Version = string(Read(buf, versionBytes))
	}

	return msg
}
