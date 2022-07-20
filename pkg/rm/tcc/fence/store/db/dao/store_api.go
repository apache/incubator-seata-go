///
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
///

package dao

import (
	"database/sql/driver"
	"time"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"
)

// The TCC Fence Store
type TCCFenceStore interface {

	// QueryTCCFenceDO tcc fence do.
	// param xid the global transaction id
	// param branchId the branch transaction id
	// return the tcc fence do
	QueryTCCFenceDO(conn driver.Conn, xid string, branchId int64) *model.TCCFenceDO

	// InsertTCCFenceDO tcc fence do boolean.
	// param tccFenceDO the tcc fence do
	// return the boolean

	InsertTCCFenceDO(conn driver.Conn, tccFenceDo model.TCCFenceDO) bool

	// UpdateTCCFenceDO tcc fence do boolean.
	// param xid the global transaction id
	// param branchId the branch transaction id
	// param newStatus the new status
	// return the boolean

	UpdateTCCFenceDO(conn driver.Conn, xid string, branchId int64, newStatus int32, oldStatus int32) bool

	// DeleteTCCFenceDO tcc fence do boolean.
	// param xid the global transaction id
	// param branchId the branch transaction id
	// return the boolean

	DeleteTCCFenceDO(conn driver.Conn, xid string, branchId int64) bool

	// DeleteTCCFenceDOByDate tcc fence by datetime.
	// param datetime datetime
	// return the deleted row count

	DeleteTCCFenceDOByDate(conn driver.Conn, datetime time.Time) bool

	// SetLogTableName LogTable Name
	// param logTableName logTableName

	SetLogTableName(logTable string)
}
