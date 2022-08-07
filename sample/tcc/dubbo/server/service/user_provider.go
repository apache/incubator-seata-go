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

	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"github.com/seata/seata-go/pkg/tm"
)

type UserProvider struct {
}

func (t *UserProvider) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	logger.Infof("Prepare result: %v, xid %v", params, tm.GetXID(ctx))
	return true, nil
}

func (t *UserProvider) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	logger.Infof("Commit result: %v, xid %s", businessActionContext, tm.GetXID(ctx))
	return true, nil
}

func (t *UserProvider) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	logger.Infof("Rollback result: %v, xid %s", businessActionContext, tm.GetXID(ctx))
	return true, nil
}

func (t *UserProvider) GetActionName() string {
	logger.Infof("GetActionName result")
	return "TwoPhaseDemoService"
}
