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
	"database/sql"
	"fmt"
	"sync"

	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/rm/tcc/fence"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

const (
	DriverName = "mysql"
	Url        = "root:root@tcp(127.0.0.1:3306)/seata?charset=utf8&parseTime=True"
)

var (
	tccService     *tcc.TCCServiceProxy
	tccServiceOnce sync.Once

	tccService2     *tcc.TCCServiceProxy
	tccService2Once sync.Once

	commitTimes      int
	commitFenceTimes int

	rollbackTimes      int
	rollbackFenceTimes int
)

type TestTCCServiceBusiness struct{}

func NewTestTCCServiceBusinessProxy() *tcc.TCCServiceProxy {
	if tccService != nil {
		return tccService
	}
	tccServiceOnce.Do(func() {
		var err error
		tccService, err = tcc.NewTCCServiceProxy(&TestTCCServiceBusiness{})
		if err != nil {
			panic(fmt.Errorf("get TestTCCServiceBusiness tcc service proxy error, %v", err.Error()))
		}
	})
	return tccService
}

func (T TestTCCServiceBusiness) Prepare(ctx context.Context, params interface{}) (b bool, err error) {
	db, err := sql.Open(DriverName, Url)
	if err != nil {
		return false, fmt.Errorf("database connect failed, msg :%s", err.Error())
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("transaction begin failed, msg :%s", err.Error())
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("business method throw error: %s, rollback result: %s", err, tx.Rollback())
			return
		}
		b, err = true, tx.Commit()
	}()

	err = fence.WithFence(ctx, tx, func() error {
		log.Infof("TestTCCServiceBusiness Prepare, param %v", params)
		return nil
	})

	return
}

func (T TestTCCServiceBusiness) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (b bool, err error) {
	db, err := sql.Open(DriverName, Url)
	if err != nil {
		return false, fmt.Errorf("database connect failed, msg :%s", err.Error())
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("transaction begin failed, msg :%s", err.Error())
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("business method throw error: %s, rollback result: %s", err, tx.Rollback())
			return
		}
		b, err = true, tx.Commit()
	}()

	err = fence.WithFence(ctx, tx, func() error {
		log.Infof("TestTCCServiceBusiness Commit, param %v", businessActionContext)
		return nil
	})

	return
}

func (T TestTCCServiceBusiness) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (b bool, err error) {
	db, err := sql.Open(DriverName, Url)
	if err != nil {
		return false, fmt.Errorf("database connect failed, msg :%s", err.Error())
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("transaction begin failed, msg :%s", err.Error())
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("business method throw error: %s, rollback result: %s", err, tx.Rollback())
			return
		}
		b, err = true, tx.Commit()
	}()

	err = fence.WithFence(ctx, tx, func() error {
		log.Infof("TestTCCServiceBusiness Rollback, param %v", businessActionContext)
		return nil
	})

	return
}

func (T TestTCCServiceBusiness) GetActionName() string {
	return "TestTCCServiceBusiness"
}

type TestTCCServiceBusiness2 struct{}

func NewTestTCCServiceBusiness2Proxy() *tcc.TCCServiceProxy {
	if tccService2 != nil {
		return tccService2
	}
	tccService2Once.Do(func() {
		var err error
		tccService2, err = tcc.NewTCCServiceProxy(&TestTCCServiceBusiness2{})
		if err != nil {
			panic(fmt.Errorf("TestTCCServiceBusiness2 get tcc service proxy error, %v", err.Error()))
		}
		if err != nil {
			panic(fmt.Errorf("TestTCCServiceBusiness2 register resource error, %v", err.Error()))
		}
	})
	return tccService2
}

func (T TestTCCServiceBusiness2) Prepare(ctx context.Context, params interface{}) (b bool, err error) {
	db, err := sql.Open(DriverName, Url)
	if err != nil {
		return false, fmt.Errorf("database connect failed, msg :%s", err.Error())
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("transaction begin failed, msg :%s", err.Error())
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("business method throw error: %s, rollback result: %s", err, tx.Rollback())
			return
		}
		b, err = true, tx.Commit()
	}()

	err = fence.WithFence(ctx, tx, func() error {
		log.Infof("TestTCCServiceBusiness2 Prepare, param %v", params)
		return nil
	})

	return
}

func (T TestTCCServiceBusiness2) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (b bool, err error) {
	db, err := sql.Open(DriverName, Url)
	if err != nil {
		return false, fmt.Errorf("database connect failed, msg :%s", err.Error())
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("transaction begin failed, msg :%s", err.Error())
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("business method throw error: %s, rollback result: %s", err, tx.Rollback())
			return
		}
		b, err = true, tx.Commit()
	}()

	err = fence.WithFence(ctx, tx, func() error {
		log.Infof("TestTCCServiceBusiness2 Commit, param %v", businessActionContext)
		return nil
	})

	return
}

func (T TestTCCServiceBusiness2) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (b bool, err error) {
	db, err := sql.Open(DriverName, Url)
	if err != nil {
		return false, fmt.Errorf("database connect failed, msg :%s", err.Error())
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("transaction begin failed, msg :%s", err.Error())
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("business method throw error: %s, rollback result: %s", err, tx.Rollback())
			return
		}
		b, err = true, tx.Commit()
	}()

	err = fence.WithFence(ctx, tx, func() error {
		log.Infof("TestTCCServiceBusiness2 Rollback, param %v", businessActionContext)
		return nil
	})

	return
}

func (T TestTCCServiceBusiness2) GetActionName() string {
	return "TestTCCServiceBusiness2"
}
