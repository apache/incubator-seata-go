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
	"seata.apache.org/seata-go/pkg/util/log"
	"strings"
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
			tccFenceStoreDatabaseMapper = &TccFenceStoreDatabaseMapper{
				logTableName: "tcc_fence_log",
			}
		})
	}
	return tccFenceStoreDatabaseMapper
}

func (t *TccFenceStoreDatabaseMapper) InitLogTableName(logTableName string) {
	// set log table name
	// default name is tcc_fence_log
	if logTableName != "" {
		t.logTableName = logTableName
	} else {
		t.logTableName = "tcc_fence_log"
	}
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
			return nil, nil
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

func (t *TccFenceStoreDatabaseMapper) QueryTCCFenceLogIdentityByMdDate(tx *sql.Tx, datetime time.Time) ([]model.FenceLogIdentity, error) {
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GetQuerySQLByMdDate(t.logTableName))
	if err != nil {
		return nil, fmt.Errorf("query tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	rows, err := prepareStmt.Query(datetime)
	if err != nil {
		return nil, fmt.Errorf("query tcc fence exec sql failed, [%w]", err)
	}
	defer rows.Close()

	var fenceLogIdentities []model.FenceLogIdentity
	for rows.Next() {
		var xid string
		var branchId int64
		err := rows.Scan(&xid, &branchId)
		if err != nil {
			return nil, fmt.Errorf("query tcc fence get scan row failed, [%w]", err)
		}
		fenceLogIdentities = append(fenceLogIdentities, model.FenceLogIdentity{
			Xid:      xid,
			BranchId: branchId,
		})
	}
	return fenceLogIdentities, nil
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

func (t *TccFenceStoreDatabaseMapper) DeleteMultipleTCCFenceLogIdentity(tx *sql.Tx, identities []model.FenceLogIdentity) error {

	placeholders := strings.Repeat("(?,?),", len(identities)-1) + "(?,?)"
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GertDeleteSQLByBranchIdsAndXids(t.logTableName, placeholders))
	if err != nil {
		return fmt.Errorf("delete tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	// prepare args
	args := make([]interface{}, 0, len(identities)*2)
	for _, identity := range identities {
		args = append(args, identity.Xid, identity.BranchId)
	}

	result, err := prepareStmt.Exec(args...)

	if err != nil {
		return fmt.Errorf("delete tcc fences exec sql failed, [%w]", err)
	}

	log.Debugf("Delete SQL: %s, args: %v", sql2.GertDeleteSQLByBranchIdsAndXids(t.logTableName, placeholders), args)

	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete tcc fences get affected rows failed, [%w]", err)
	}

	return nil
}

func (t *TccFenceStoreDatabaseMapper) DeleteTCCFenceDOByMdfDate(tx *sql.Tx, datetime time.Time, limit int32) (int64, error) {
	prepareStmt, err := tx.PrepareContext(context.Background(), sql2.GetDeleteSQLByMdfDateAndStatus(t.logTableName))
	if err != nil {
		return -1, fmt.Errorf("delete tcc fence prepare sql failed, [%w]", err)
	}
	defer prepareStmt.Close()

	result, err := prepareStmt.Exec(datetime, limit)
	if err != nil {
		return -1, fmt.Errorf("delete tcc fence exec sql failed, [%w]", err)
	}

	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return 0, fmt.Errorf("delete tcc fence get affected rows failed, [%w]", err)
	}

	return affected, nil
}

func (t *TccFenceStoreDatabaseMapper) SetLogTableName(logTable string) {
	t.logTableName = logTable
}
