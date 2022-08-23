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
	"time"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/local/service"
)

func main() {
	client.Init()
	var err error
	ctx := tm.Begin(context.Background(), "TestTCCServiceBusiness")
	timer := time.NewTimer(15 * time.Second)
	defer func() {
		<-time.After(15 * time.Second)
		ok := timer.Stop()
		if ok {
			resp := tm.CommitOrRollback(ctx, err == nil)
			log.Infof("tx result %v", resp)
			<-make(chan struct{})
		}
	}()

	tccService := service.NewTestTCCServiceBusinessProxy()
	tccService2 := service.NewTestTCCServiceBusiness2Proxy()

	_, err = tccService.Prepare(ctx, 1)
	if err != nil {
		log.Errorf("TestTCCServiceBusiness prepare error, %v", err.Error())
		return
	}
	_, err = tccService2.Prepare(ctx, 3)
	if err != nil {
		log.Errorf("TestTCCServiceBusiness2 prepare error, %v", err.Error())
		return
	}

}
