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

package main

import (
	"context"

	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/integration"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/dubbo/client/service"
)

// need to setup environment variable "DUBBO_GO_CONFIG_PATH" to "conf/dubbogo.yaml" before run
func main() {
	integration.UseDubbo()
	config.SetConsumerService(service.UserProviderInstance)
	err := config.Load()
	if err != nil {
		panic(err)
	}
	test()
}

func test() {
	var err error
	ctx := tm.Begin(context.Background(), "TestTCCServiceBusiness")
	defer func() {
		resp := tm.CommitOrRollback(ctx, &err)
		logger.Infof("tx result %v", resp)
		<-make(chan struct{})
	}()

	userProviderProxy, err := tcc.NewTCCServiceProxy(service.UserProviderInstance)
	if err != nil {
		logger.Infof("userProviderProxyis not tcc service")
		return
	}
	resp, err := userProviderProxy.Prepare(ctx, 1)
	if err != nil {
		logger.Infof("response prepare: %v", err)
		return
	}
	logger.Infof("get resp %#v", resp)
}
