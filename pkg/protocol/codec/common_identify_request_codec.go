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

type AbstractIdentifyRequestCodec struct{}

func (c *AbstractIdentifyRequestCodec) Encode(in interface{}) []byte {
	data := in.(message.AbstractIdentifyRequest)
	buf := bytes.NewByteBuffer([]byte{})

	bytes.WriteString16Length(data.Version, buf)
	bytes.WriteString16Length(data.ApplicationId, buf)
	bytes.WriteString16Length(data.TransactionServiceGroup, buf)
	bytes.WriteString16Length(string(data.ExtraData), buf)

	return buf.Bytes()
}

func (c *AbstractIdentifyRequestCodec) Decode(in []byte) interface{} {
	data := message.AbstractIdentifyRequest{}
	buf := bytes.NewByteBuffer(in)

	data.Version = bytes.ReadString16Length(buf)
	data.ApplicationId = bytes.ReadString16Length(buf)
	data.TransactionServiceGroup = bytes.ReadString16Length(buf)
	data.ExtraData = []byte(bytes.ReadString16Length(buf))

	return data
}
