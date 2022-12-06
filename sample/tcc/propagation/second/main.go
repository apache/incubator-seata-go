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

package second

import (
	"context"
	"fmt"
	"sync"

	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

var (
	tccService     *tcc.TCCServiceProxy
	tccServiceOnce sync.Once
)

type TestTccServiceBusiness struct{}

func NewTccServiceProxy() *tcc.TCCServiceProxy {
	if tccService != nil {
		return tccService
	}
	tccServiceOnce.Do(func() {
		var err error
		tccService, err = tcc.NewTCCServiceProxy(&TestTccServiceBusiness{})
		if err != nil {
			panic(fmt.Errorf("get TestTccServiceBusiness tcc service proxy error, %v", err.Error()))
		}
	})
	return tccService
}

func (T TestTccServiceBusiness) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("SecondPrepare: propagation tx %s/%s, case to rollback", tm.GetTxName(ctx), tm.GetXID(ctx))
	return false, fmt.Errorf("SecondPrepare: propagation tx %s/%s, case to rollback", tm.GetTxName(ctx), tm.GetXID(ctx))
}

func (T TestTccServiceBusiness) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("SecondCommit: propagation tx xid %v", tm.GetXID(ctx))
	return true, nil
}

func (T TestTccServiceBusiness) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("SecondRollback: propagation tx xid %v", tm.GetXID(ctx))
	return true, nil
}

func (T TestTccServiceBusiness) GetActionName() string {
	return "TestTCCServiceBusinessSecond"
}

func Business(ctx context.Context) (re error) {
	log.Infof("SecondBusiness: propagation tx xid %v", tm.GetXID(ctx))
	if _, re = NewTccServiceProxy().Prepare(ctx, 1); re != nil {
		log.Errorf("SecondBusiness prepare error, %v", re)
		return
	}

	return
}
