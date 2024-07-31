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

package dao

import (
	"database/sql"
	"time"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/model"
)

// The TCC Fence Store
type TCCFenceStore interface {

	// QueryTCCFenceDO tcc fence do.
	// param tx the tx will bind with user business method
	// param xid the global transaction id
	// param branchId the branch transaction id
	// return the tcc fence do and error msg
	QueryTCCFenceDO(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error)

	// InsertTCCFenceDO tcc fence do boolean.
	// param tx the tx will bind with user business method
	// param tccFenceDO the tcc fence do
	// return the error msg
	InsertTCCFenceDO(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error

	// UpdateTCCFenceDO tcc fence do boolean.
	// param tx the tx will bind with user business method
	// param xid the global transaction id
	// param branchId the branch transaction id
	// param newStatus the new status
	// return the error msg
	UpdateTCCFenceDO(tx *sql.Tx, xid string, branchId int64, oldStatus enum.FenceStatus, newStatus enum.FenceStatus) error

	// DeleteTCCFenceDO tcc fence do boolean.
	// param tx the tx will bind with user business method
	// param xid the global transaction id
	// param branchId the branch transaction id
	// return the error msg
	DeleteTCCFenceDO(tx *sql.Tx, xid string, branchId int64) error

	// DeleteTCCFenceDOByMdfDate tcc fence by datetime.
	// param tx the tx will bind with user business method
	// param datetime modify time
	// return the error msg
	DeleteTCCFenceDOByMdfDate(tx *sql.Tx, datetime time.Time) error

	// SetLogTableName LogTable ColumnName
	// param logTableName logTableName
	SetLogTableName(logTable string)
}
