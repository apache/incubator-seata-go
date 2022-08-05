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

	"github.com/seata/seata-go/pkg/common/errors"

	"strings"
	"sync"
	"time"

	sql2 "github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/sql"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"
)

var (
	once                        sync.Once
	tccFenceStoreDatabaseMapper *TccFenceStoreDatabaseMapper
)

func GetTccFenceStoreDatabaseMapperSingleton() *TccFenceStoreDatabaseMapper {
	if tccFenceStoreDatabaseMapper == nil {
		once.Do(func() {
			tccFenceStoreDatabaseMapper = &TccFenceStoreDatabaseMapper{}
			tccFenceStoreDatabaseMapper.logTableName = "tcc_fence_log"
		})
	}
	return tccFenceStoreDatabaseMapper

}

type TccFenceStoreDatabaseMapper struct {
	logTableName string
}

func (tcs *TccFenceStoreDatabaseMapper) QueryTCCFenceDO(tx *sql.Tx, xid string, branchId int64) *model.TCCFenceDO {
	var prepareStmt *sql.Stmt
	var tccFenceDo *model.TCCFenceDO
	sql := sql2.GetQuerySQLByBranchIdAndXid(tcs.logTableName)
	prepareStmt, err := tx.PrepareContext(context.Background(), sql)
	if err != nil {
		panic(fmt.Sprintf("query tcc fence prepare sql failed msg : %v", err))
	}
	defer prepareStmt.Close()
	result := prepareStmt.QueryRow(xid, branchId)
	var (
		actionName string
		status     int32
		gmtCreate  time.Time
		gmtModify  time.Time
	)

	if errScan := result.Scan(&xid, &branchId, &actionName, &status, &gmtCreate, &gmtModify); errScan == nil {
		tccFenceDo = &model.TCCFenceDO{
			Xid:         xid,
			BranchId:    branchId,
			ActionName:  actionName,
			Status:      status,
			GmtModified: gmtModify,
			GmtCreate:   gmtCreate,
		}
	} else {
		panic(fmt.Sprintf("query tcc fence get scan row failed msg : %v", errScan))
	}
	return tccFenceDo
}

func (tcs *TccFenceStoreDatabaseMapper) InsertTCCFenceDO(tx *sql.Tx, tccFenceDo *model.TCCFenceDO) bool {
	var prepareStmt *sql.Stmt
	timeNow := time.Now()
	sql := sql2.GetInsertLocalTCCLogSQL(tcs.logTableName)
	prepareStmt, err := tx.PrepareContext(context.Background(), sql)

	if err != nil {
		panic(fmt.Sprintf("insert tcc fence prepare sql failed msg : %v", err))
	}
	defer prepareStmt.Close()

	result, errStmt := prepareStmt.Exec(tccFenceDo.Xid, tccFenceDo.BranchId, tccFenceDo.ActionName, tccFenceDo.Status, timeNow, timeNow)
	if errStmt == nil {
		if affected, errAff := result.RowsAffected(); errAff == nil {
			return affected > 0
		} else {
			panic(fmt.Sprintf("insert tcc fence get rows affected failed msg : %v", errAff))
		}
	} else {
		if strings.Contains(errStmt.Error(), "Error 1062: Duplicate entry") {
			panic(errors.NewTccFenceError(errors.FenceErrorCodeDuplicateKey, fmt.Sprintf("Insert tcc fence record duplicate key exception. xid= %s, branchId= %d", tccFenceDo.Xid, tccFenceDo.BranchId)))
		} else {
			panic(fmt.Errorf("insert tcc fence execute sql failed msg : %v", errStmt))
		}
	}
}

func (tcs *TccFenceStoreDatabaseMapper) UpdateTCCFenceDO(tx *sql.Tx, xid string, branchId int64, oldStatus int32, newStatus int32) bool {
	var prepareStmt *sql.Stmt
	timeNow := time.Now()
	sql := sql2.GetUpdateStatusSQLByBranchIdAndXid(tcs.logTableName)
	prepareStmt, err := tx.PrepareContext(context.Background(), sql)

	if err != nil {
		panic(fmt.Errorf("update tcc fence prepare sql failed msg : %v", err))
	}
	defer prepareStmt.Close()

	result, errStmt := prepareStmt.Exec(newStatus, timeNow, xid, branchId, oldStatus)
	if errStmt == nil {
		if affected, errAff := result.RowsAffected(); errAff == nil {
			return affected > 0
		} else {
			panic(fmt.Errorf("update tcc fence get rows affected failed msg : %v", errAff))
		}
	} else {
		panic(fmt.Errorf("update tcc fence execute sql failed msg : %v", errStmt))
	}
}

func (tcs *TccFenceStoreDatabaseMapper) DeleteTCCFenceDO(tx *sql.Tx, xid string, branchId int64) bool {
	var prepareStmt *sql.Stmt
	sql := sql2.GetDeleteSQLByBranchIdAndXid(tcs.logTableName)
	prepareStmt, err := tx.PrepareContext(context.Background(), sql)

	if err != nil {
		panic(fmt.Errorf("delete tcc fence prepare sql failed msg : %v", err))
	}
	defer prepareStmt.Close()

	result, errStmt := prepareStmt.Exec(xid, branchId)
	if errStmt == nil {
		if affected, errAff := result.RowsAffected(); errAff == nil {
			return affected > 0
		} else {
			panic(fmt.Sprintf("delete tcc fence get rows affected failed msg : %v", errAff))
		}
	} else {
		panic(fmt.Sprintf("delete tcc fence execute sql failed msg : %v", errStmt))
	}
}

func (tcs *TccFenceStoreDatabaseMapper) DeleteTCCFenceDOByMdfDate(tx *sql.Tx, datetime time.Time) bool {
	var prepareStmt *sql.Stmt
	sql := sql2.GetDeleteSQLByMdfDateAndStatus(tcs.logTableName)
	prepareStmt, err := tx.PrepareContext(context.Background(), sql)

	if err != nil {
		panic(fmt.Errorf("delete tcc fence prepare sql failed msg : %v", err))
	}
	defer prepareStmt.Close()

	result, errStmt := prepareStmt.Exec(datetime)
	if errStmt == nil {
		if affected, errAff := result.RowsAffected(); errAff == nil {
			return affected > 0
		} else {
			panic(fmt.Sprintf("delete tcc fence get rows affected failed msg : %v", errAff))
		}
	} else {
		panic(fmt.Sprintf("delete tcc fence execute sql failed msg : %v", errStmt))
	}
}

func (tcs *TccFenceStoreDatabaseMapper) SetLogTableName(logTable string) {
	tcs.logTableName = logTable
}
