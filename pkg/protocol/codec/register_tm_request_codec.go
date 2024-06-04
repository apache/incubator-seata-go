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
)

type RegisterTMRequestCodec struct {
	AbstractIdentifyRequestCodec
}

func (g *RegisterTMRequestCodec) Decode(in []byte) interface{} {
	req := g.AbstractIdentifyRequestCodec.Decode(in)
	abstractIdentifyRequest := req.(message.AbstractIdentifyRequest)
	return message.RegisterTMRequest{
		AbstractIdentifyRequest: abstractIdentifyRequest,
	}
}

func (c *RegisterTMRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.RegisterTMRequest)
	return c.AbstractIdentifyRequestCodec.Encode(req.AbstractIdentifyRequest)
}

func (g *RegisterTMRequestCodec) GetMessageType() message.MessageType {
	return message.MessageTypeRegClt
}
