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

type GlobalReportResponseCodec struct {
	CommonGlobalEndResponseCodec
}

func (g *GlobalReportResponseCodec) Decode(in []byte) interface{} {
	req := g.CommonGlobalEndResponseCodec.Decode(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndResponse)
	return message.GlobalReportResponse{
		AbstractGlobalEndResponse: abstractGlobalEndRequest,
	}
}

func (g *GlobalReportResponseCodec) Encode(in interface{}) []byte {
	req := in.(message.GlobalReportResponse)
	return g.CommonGlobalEndResponseCodec.Encode(req.AbstractGlobalEndResponse)
}

func (g *GlobalReportResponseCodec) GetMessageType() message.MessageType {
	return message.MessageTypeGlobalReportResult
}
