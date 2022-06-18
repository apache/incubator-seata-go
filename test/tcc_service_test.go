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
	"context"
	"testing"
	"time"

	"github.com/seata/seata-go/pkg/tm"

	"github.com/seata/seata-go/pkg/common/log"

	_ "github.com/seata/seata-go/pkg/imports"

	"github.com/seata/seata-go/pkg/rm/tcc"
)

type TestTCCServiceBusiness struct {
}

func (T TestTCCServiceBusiness) Prepare(ctx context.Context, params interface{}) error {
	log.Infof("TestTCCServiceBusiness Prepare, param %v", params)
	return nil
}

func (T TestTCCServiceBusiness) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Commit, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Rollback, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness) GetActionName() string {
	return "TestTCCServiceBusiness"
}

type TestTCCServiceBusiness2 struct {
}

func (T TestTCCServiceBusiness2) Prepare(ctx context.Context, params interface{}) error {
	log.Infof("TestTCCServiceBusiness2 Prepare, param %v", params)
	return nil
}

func (T TestTCCServiceBusiness2) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness2 Commit, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness2) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness2 Rollback, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness2) GetActionName() string {
	return "TestTCCServiceBusiness2"
}

func TestNew(test *testing.T) {
	var err error
	ctx := tm.Begin(context.Background(), "TestTCCServiceBusiness")
	defer func() {
		resp := tm.CommitOrRollback(ctx, err)
		log.Infof("tx result %v", resp)
	}()

	tccService := tcc.NewTCCServiceProxy(TestTCCServiceBusiness{})
	err = tccService.Prepare(ctx, 1)
	if err != nil {
		log.Errorf("execute TestTCCServiceBusiness prepare error %s", err.Error())
		return
	}

	tccService2 := tcc.NewTCCServiceProxy(TestTCCServiceBusiness2{})
	err = tccService2.Prepare(ctx, 3)
	if err != nil {
		log.Errorf("execute TestTCCServiceBusiness2 prepare error %s", err.Error())
		return
	}

	time.Sleep(time.Second * 1000)
}
