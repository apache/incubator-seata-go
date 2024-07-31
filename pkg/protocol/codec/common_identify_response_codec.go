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

type AbstractIdentifyResponseCodec struct{}

func (c *AbstractIdentifyResponseCodec) Encode(in interface{}) []byte {
	data := in.(message.AbstractIdentifyResponse)
	buf := bytes.NewByteBuffer([]byte{})

	if data.Identified {
		buf.WriteByte(byte(1))
	} else {
		buf.WriteByte(byte(0))
	}
	bytes.WriteString16Length(data.Version, buf)

	return buf.Bytes()
}

func (c *AbstractIdentifyResponseCodec) Decode(in []byte) interface{} {
	data := message.AbstractIdentifyResponse{}
	buf := bytes.NewByteBuffer(in)

	identified, _ := buf.ReadByte()
	if identified == byte(1) {
		data.Identified = true
	} else if identified == byte(0) {
		data.Identified = false
	}
	data.Version = bytes.ReadString16Length(buf)

	return data
}
