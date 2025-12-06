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

package getty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
)

func TestRpcPackageHandler(t *testing.T) {
	msg := message.RpcMessage{
		ID:         1123,
		Type:       message.GettyRequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: byte(1),
		HeadMap: map[string]string{
			"name":    " Jack",
			"age":     "12",
			"address": "Beijing",
		},
		Body: message.GlobalBeginRequest{
			Timeout:         2 * time.Second,
			TransactionName: "SeataGoTransaction",
		},
	}

	codec := RpcPackageHandler{}
	bytes, err := codec.Write(nil, msg)
	assert.Nil(t, err)
	msg2, _, _ := codec.Read(nil, bytes)

	assert.Equal(t, msg, msg2)
}
