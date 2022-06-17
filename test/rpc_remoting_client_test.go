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

package test

import (
	"testing"

	_ "github.com/seata/seata-go/pkg/imports"
)

func TestSendMsgWithResponse(test *testing.T) {
	//request := protocol.RegisterRMRequest{
	//	ResourceIds: "1111",
	//	AbstractIdentifyRequest: protocol.AbstractIdentifyRequest{
	//		ApplicationId:           "ApplicationID",
	//		TransactionServiceGroup: "TransactionServiceGroup",
	//	},
	//}
	//mergedMessage := protocol.MergedWarpMessage{
	//	Msgs:   []protocol.MessageTypeAware{request},
	//	MsgIds: []int32{1212},
	//}
	//handler := GetGettyClientHandlerInstance()
	//handler.sendMergedMessage(mergedMessage)
	//time.Sleep(100000 * time.Second)
}
