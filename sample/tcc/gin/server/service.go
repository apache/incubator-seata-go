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

	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

type RMService struct{}

func (b *RMService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("TRMService Prepare, param %v", params)
	return true, nil
}

func (b *RMService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("RMService Commit, param %v", businessActionContext)
	return true, nil
}

func (b *RMService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("RMService Rollback, param %v", businessActionContext)
	return true, nil
}

func (b *RMService) GetActionName() string {
	return "ginTccRMService"
}
