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
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/model"
	sql2 "seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/sql"
)

var (
	once                        sync.Once
	tccFenceStoreDatabaseMapper *TccFenceStoreDatabaseMapper
)

func GetTccFenceStoreDatabaseMapper() *TccFenceStoreDatabaseMapper {
	if tccFenceStoreDatabaseMapper == nil {
		once.Do(func() {
			tccFenceStoreDatabaseMapper = &TccFenceStoreDatabaseMapper{}
			tccFenceStoreDatabaseMapper.InitLogTableName()
		})
	}
	return tccFenceStoreDatabaseMapper
}

func (t *TccFenceStoreDatabaseMapper) InitLogTableName() {
	// todo get log table name from config
	// set log table name
	// default name is tcc_fence_log
	t.logTableName = "tcc_fence_log"
}

type TccFenceStoreDatabaseMapper struct {
	logTableName string
}

func (t *TccFenceStoreDatabaseMapper) QueryTCCFenceDO(tx *sql.Tx, xid string, branchId int64) (*model.TCCFenceDO, error) {
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GetQuerySQLByBranchIdAndXid(t.logTableName))
	if err != nil {
		return nil, fmt.Errorf("query tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	result := prepareStmt.QueryRow(xid, branchId)
	var (
		actionName string
		status     enum.FenceStatus
		gmtCreate  time.Time
		gmtModify  time.Time
	)

	if err = result.Scan(&xid, &branchId, &actionName, &status, &gmtCreate, &gmtModify); err != nil {
		// will return error, if rows is empty
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("query tcc fence get scan row，no rows in result set, [%w]", err)
		} else {
			return nil, fmt.Errorf("query tcc fence get scan row failed, [%w]", err)
		}
	}

	tccFenceDo := &model.TCCFenceDO{
		Xid:         xid,
		BranchId:    branchId,
		ActionName:  actionName,
		Status:      status,
		GmtModified: gmtModify,
		GmtCreate:   gmtCreate,
	}
	return tccFenceDo, nil
}

func (t *TccFenceStoreDatabaseMapper) InsertTCCFenceDO(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) error {
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GetInsertLocalTCCLogSQL(t.logTableName))
	if err != nil {
		return fmt.Errorf("insert tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	timeNow := time.Now()
	result, err := prepareStmt.Exec(tccFenceDo.Xid, tccFenceDo.BranchId, tccFenceDo.ActionName, tccFenceDo.Status, timeNow, timeNow)
	if err != nil {
		if mysqlError, ok := err.(*mysql.MySQLError); ok && mysqlError.Number == 1062 {
			return fmt.Errorf("insert tcc fence record duplicate key exception. xid= %s, branchId= %d, [%w]", tccFenceDo.Xid, tccFenceDo.BranchId, err)
		} else {
			return fmt.Errorf("insert tcc fence exec sql failed, [%w]", err)
		}
	}

	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return fmt.Errorf("insert tcc fence get affected rows failed, [%w]", err)
	}

	return nil
}

func (t *TccFenceStoreDatabaseMapper) UpdateTCCFenceDO(tx *sql.Tx, xid string, branchId int64, oldStatus enum.FenceStatus, newStatus enum.FenceStatus) error {
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GetUpdateStatusSQLByBranchIdAndXid(t.logTableName))
	if err != nil {
		return fmt.Errorf("update tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	result, err := prepareStmt.Exec(newStatus, time.Now(), xid, branchId, oldStatus)
	if err != nil {
		return fmt.Errorf("update tcc fence exec sql failed, [%w]", err)
	}

	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return fmt.Errorf("update tcc fence get affected rows failed, [%w]", err)
	}

	return nil
}

func (t *TccFenceStoreDatabaseMapper) DeleteTCCFenceDO(tx *sql.Tx, xid string, branchId int64) error {
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GetDeleteSQLByBranchIdAndXid(t.logTableName))
	if err != nil {
		return fmt.Errorf("delete tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	result, err := prepareStmt.Exec(xid, branchId)
	if err != nil {
		return fmt.Errorf("delete tcc fence exec sql failed, [%w]", err)
	}

	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return fmt.Errorf("delete tcc fence get affected rows failed, [%w]", err)
	}

	return nil
}

func (t *TccFenceStoreDatabaseMapper) DeleteTCCFenceDOByMdfDate(tx *sql.Tx, datetime time.Time) error {
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GetDeleteSQLByMdfDateAndStatus(t.logTableName))
	if err != nil {
		return fmt.Errorf("delete tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	result, err := prepareStmt.Exec(datetime)
	if err != nil {
		return fmt.Errorf("delete tcc fence exec sql failed, [%w]", err)
	}

	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return fmt.Errorf("delete tcc fence get affected rows failed, [%w]", err)
	}

	return nil
}

func (t *TccFenceStoreDatabaseMapper) SetLogTableName(logTable string) {
	t.logTableName = logTable
}
