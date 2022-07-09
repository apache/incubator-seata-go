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

	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/local/service"
)

func main() {

	var err error
	ctx := tm.Begin(context.Background(), "TestTCCServiceBusiness")
	defer func() {
		resp := tm.CommitOrRollback(ctx, &err)
		log.Infof("tx result %v", resp)
		<-make(chan struct{})
	}()

	tccService, err := tcc.NewTCCServiceProxy(&service.TestTCCServiceBusiness{})
	if err != nil {
		log.Errorf("get TestTCCServiceBusiness tcc service proxy error, %v", err.Error())
		return
	}
	err = tccService.RegisterResource()
	if err != nil {
		log.Errorf("TestTCCServiceBusiness register resource error, %v", err.Error())
		return
	}
	_, err = tccService.Prepare(ctx, 1)
	if err != nil {
		log.Errorf("TestTCCServiceBusiness prepare error, %v", err.Error())
		return
	}

	tccService2, err := tcc.NewTCCServiceProxy(&service.TestTCCServiceBusiness2{})
	if err != nil {
		log.Errorf("get TestTCCServiceBusiness2 tcc service proxy error, %v", err.Error())
		return
	}
	err = tccService2.RegisterResource()
	if err != nil {
		log.Errorf("TestTCCServiceBusiness2 register resource error, %v", err.Error())
		return
	}
	_, err = tccService2.Prepare(ctx, 3)
	if err != nil {
		log.Errorf("TestTCCServiceBusiness2 prepare error, %v", err.Error())
		return
	}

}
