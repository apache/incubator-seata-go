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
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/message"
)

func TestRegisterTMResponseCodec(t *testing.T) {
	msg := message.RegisterTMResponse{
		AbstractIdentifyResponse: message.AbstractIdentifyResponse{
			AbstractResultMessage: message.AbstractResultMessage{
				ResultCode: message.ResultCodeFailed,
				Msg:        "TestMsg",
			},
			ExtraData:  []byte("TestExtraData"),
			Version:    "V1,0",
			Identified: false,
		},
	}

	codec := RegisterTMResponseCodec{}
	bytes := codec.Encode(msg)
	msg2 := codec.Decode(bytes)

	assert.Equal(t, msg.Identified, msg2.(message.RegisterTMResponse).Identified)
	assert.Equal(t, msg.Version, msg2.(message.RegisterTMResponse).Version)
}
