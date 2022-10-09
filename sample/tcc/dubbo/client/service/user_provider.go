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
	"fmt"

	"github.com/seata/seata-go/pkg/tm"
)

var UserProviderInstance = NewTwoPhaseDemoService()

type UserProvider struct {
	Prepare       func(ctx context.Context, params ...interface{}) (bool, error)                           `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
	Commit        func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
	Rollback      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
	GetActionName func() string
}

func NewTwoPhaseDemoService() *UserProvider {
	return &UserProvider{
		Prepare: func(ctx context.Context, params ...interface{}) (bool, error) {
			return false, fmt.Errorf("execute two phase prepare method, param %v", params)
		},
		Commit: func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
			return false, fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
		},
		Rollback: func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
			return true, nil
		},
		GetActionName: func() string {
			return "TwoPhaseDemoService"
		},
	}
}
