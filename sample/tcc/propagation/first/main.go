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
	"fmt"
	"sync"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	"github.com/seata/seata-go/sample/tcc/propagation/second"
)

func main() {
	client.Init()
	log.Info(tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name: "TccSampleLocalGlobalTxFirst",
	}, business))
	<-make(chan struct{})
}

func business(ctx context.Context) (re error) {
	log.Infof("FirstBusiness: propagation tx xid %v", tm.GetXID(ctx))
	if _, re = New().Prepare(ctx, 1); re != nil {
		log.Errorf("FirstBusiness prepare error, %v", re)
		return
	}
	return
}

var (
	tccService     *tcc.TCCServiceProxy
	tccServiceOnce sync.Once
)

type TestTCCServiceBusiness struct{}

func New() *tcc.TCCServiceProxy {
	if tccService != nil {
		return tccService
	}
	tccServiceOnce.Do(func() {
		var err error
		tccService, err = tcc.NewTCCServiceProxy(&TestTCCServiceBusiness{})
		if err != nil {
			panic(fmt.Errorf("get TestTccServiceBusiness tcc service proxy error, %v", err.Error()))
		}
	})
	return tccService
}

func (T TestTCCServiceBusiness) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("FirstPrepare: propagation tx xid %v", tm.GetXID(ctx))
	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:        "TccSampleLocalGlobalTxSecond",
		Propagation: tm.Mandatory,
	}, second.Business)
	return err == nil, err
}

func (T TestTCCServiceBusiness) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("FirstCommit: propagation tx xid %v", tm.GetXID(ctx))
	return true, nil
}

func (T TestTCCServiceBusiness) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("FirstRollback: propagation tx xid %v", tm.GetXID(ctx))
	return true, nil
}

func (T TestTCCServiceBusiness) GetActionName() string {
	return "TestTCCServiceBusinessFirst"
}
