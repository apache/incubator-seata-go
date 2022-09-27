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

	"github.com/seata/seata-go/pkg/client"

	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/dubbo/client/service"
)

// need to setup environment variable "DUBBO_GO_CONFIG_PATH" to "conf/dubbogo.yml" before run
func main() {
	client.Init()
	config.SetConsumerService(service.UserProviderInstance)
	if err := config.Load(); err != nil {
		panic(err)
	}
	run()
}

func run() {
	tm.WithGlobalTx(context.Background(), &tm.TransactionInfo{
		Name: "TccSampleLocalGlobalTx",
	}, business)
	<-make(chan struct{})
}

func business(ctx context.Context) (re error) {
	if resp, re := service.UserProviderInstance.Prepare(ctx, 1); re != nil {
		logger.Infof("response prepare: %v", re)
	} else {
		logger.Infof("get resp %#v", resp)
	}
	return
}
