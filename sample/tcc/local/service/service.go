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

package service

import (
	"context"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/tm"
)

type TestTCCServiceBusiness struct {
}

func (T TestTCCServiceBusiness) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	log.Infof("TestTCCServiceBusiness Prepare, param %v", params)
	return true, nil
}

func (T TestTCCServiceBusiness) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness Commit, param %v", businessActionContext)
	return true, nil
}

func (T TestTCCServiceBusiness) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness Rollback, param %v", businessActionContext)
	return true, nil
}

func (T TestTCCServiceBusiness) GetActionName() string {
	return "TestTCCServiceBusiness"
}

type TestTCCServiceBusiness2 struct {
}

func (T TestTCCServiceBusiness2) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	log.Infof("TestTCCServiceBusiness2 Prepare, param %v", params)
	return true, nil
}

func (T TestTCCServiceBusiness2) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness2 Commit, param %v", businessActionContext)
	return true, nil
}

func (T TestTCCServiceBusiness2) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness2 Rollback, param %v", businessActionContext)
	return true, nil
}

func (T TestTCCServiceBusiness2) GetActionName() string {
	return "TestTCCServiceBusiness2"
}
