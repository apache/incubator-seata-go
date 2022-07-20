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

package config

import (
	"database/sql/driver"
	"math"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/dao"

	"go.uber.org/atomic"
)

var (
	schedulerDone sync.WaitGroup
)

const (
	MaxPeriod = time.Duration(math.MaxInt32)
)

type TccFenceConfig struct {
	Initialized  atomic.Bool `default:"false"`
	LogTableName string      `default:"tcc_fence_log"`
}

func SetTccFenceConfig(datasource driver.Connector, transactionManager interface{}) {

}

func init() {
	// todo
}

func (this *TccFenceConfig) Init() {
	// set log table name
	if this.LogTableName != "" {
		dao.GetTccFenceStoreDatabaseMapperSingleton().SetLogTableName(this.LogTableName)
	}
}

func TccFenceCleanScheduler() {

}

func InitCleanTask() {
	schedulerDone.Add(1)
}

func Destory() {
	schedulerDone.Done()
}
