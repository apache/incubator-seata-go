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

type GlobalCommitRequestCodec struct {
	CommonGlobalEndRequestCodec
}

func (g *GlobalCommitRequestCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndRequestCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
	return message.GlobalCommitRequest{
		AbstractGlobalEndRequest: abstractGlobalEndRequest,
	}
}

func (g *GlobalCommitRequestCodec) Encode(in interface{}) []byte {
	req := in.(message.GlobalCommitRequest)
	return g.CommonGlobalEndRequestCodec.Encode(req.AbstractGlobalEndRequest)
}

func (g *GlobalCommitRequestCodec) GetMessageType() message.MessageType {
	return message.MessageTypeGlobalCommit
}
