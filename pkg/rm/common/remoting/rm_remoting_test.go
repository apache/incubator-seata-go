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

package remoting

import (
	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/resource"
	"github.com/seata/seata-go/pkg/remoting/getty"
)

func (RMRemoting) RegisterResource(resource resource.Resource) error {
	req := message.RegisterRMRequest{
		AbstractIdentifyRequest: message.AbstractIdentifyRequest{
			//todo replace with config
			Version:                 "1.4.2",
			ApplicationId:           "tcc-sample",
			TransactionServiceGroup: "my_test_tx_group",
		},
		ResourceIds: resource.GetResourceId(),
	}
	err := getty.GetGettyRemotingClient().SendAsyncRequest(req)
	if err != nil {
		log.Error("RegisterResourceManager error: {%#v}", err.Error())
		return err
	}
	return nil
}
