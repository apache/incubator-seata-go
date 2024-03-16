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
	"database/sql/driver"
	"math"
	"reflect"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/model"
	sql2 "seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/sql"
)

func TestTccFenceStoreDatabaseMapper_SetLogTableName(t *testing.T) {
	GetTccFenceStoreDatabaseMapper().SetLogTableName("tcc_fence_log")
	value := reflect.ValueOf(GetTccFenceStoreDatabaseMapper())
	assert.Equal(t, "tcc_fence_log", value.Elem().FieldByName("logTableName").String())
}

func TestTccFenceStoreDatabaseMapper_InsertTCCFenceDO(t *testing.T) {
	tccFenceDo := &model.TCCFenceDO{
		Xid:        "123123124124",
		BranchId:   12312312312,
		ActionName: "fence_test",
		Status:     enum.StatusSuspended,
	}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectPrepare(sql2.GetInsertLocalTCCLogSQL("tcc_fence_log")).
		ExpectExec().
		WithArgs(driver.Value(tccFenceDo.Xid), driver.Value(tccFenceDo.BranchId), driver.Value(tccFenceDo.ActionName),
			driver.Value(tccFenceDo.Status), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}

	err = GetTccFenceStoreDatabaseMapper().InsertTCCFenceDO(tx, tccFenceDo)
	tx.Commit()
	assert.Equal(t, nil, err)
}

func TestTccFenceStoreDatabaseMapper_QueryTCCFenceDO(t *testing.T) {
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		Xid:         "123123124124",
		BranchId:    12312312312,
		ActionName:  "fence_test",
		Status:      enum.StatusTried,
		GmtCreate:   now,
		GmtModified: now,
	}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare(sql2.GetQuerySQLByBranchIdAndXid("tcc_fence_log")).
		ExpectQuery().
		WithArgs(driver.Value(tccFenceDo.Xid), driver.Value(tccFenceDo.BranchId)).
		WillReturnRows(sqlmock.NewRows([]string{"xid", "branch_id", "action_name", "status", "gmt_create", "gmt_modified"}).
			AddRow(driver.Value(tccFenceDo.Xid), driver.Value(tccFenceDo.BranchId), driver.Value(tccFenceDo.ActionName),
				driver.Value(tccFenceDo.Status), driver.Value(now), driver.Value(now)))
	mock.ExpectCommit()
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}

	actualFenceDo, err := GetTccFenceStoreDatabaseMapper().QueryTCCFenceDO(tx, tccFenceDo.Xid, tccFenceDo.BranchId)
	tx.Commit()
	assert.Equal(t, tccFenceDo.Xid, actualFenceDo.Xid)
	assert.Equal(t, tccFenceDo.BranchId, actualFenceDo.BranchId)
	assert.Equal(t, tccFenceDo.Status, actualFenceDo.Status)
	assert.Equal(t, tccFenceDo.ActionName, actualFenceDo.ActionName)
	assert.NotEmpty(t, actualFenceDo.GmtModified)
	assert.NotEmpty(t, actualFenceDo.GmtCreate)
	assert.Nil(t, err)
}

func TestTccFenceStoreDatabaseMapper_UpdateTCCFenceDO(t *testing.T) {
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		Xid:         "123123124124",
		BranchId:    12312312312,
		ActionName:  "fence_test",
		Status:      enum.StatusTried,
		GmtCreate:   now,
		GmtModified: now,
	}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectPrepare(sql2.GetUpdateStatusSQLByBranchIdAndXid("tcc_fence_log")).
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}

	err = GetTccFenceStoreDatabaseMapper().
		UpdateTCCFenceDO(tx, tccFenceDo.Xid, tccFenceDo.BranchId, tccFenceDo.Status, enum.StatusCommitted)
	tx.Commit()
	assert.Equal(t, nil, err)
}

func TestTccFenceStoreDatabaseMapper_DeleteTCCFenceDO(t *testing.T) {
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		Xid:         "123123124124",
		BranchId:    12312312312,
		ActionName:  "fence_test",
		Status:      enum.StatusTried,
		GmtCreate:   now,
		GmtModified: now,
	}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare(sql2.GetDeleteSQLByBranchIdAndXid("tcc_fence_log")).
		ExpectExec().
		WithArgs(driver.Value(tccFenceDo.Xid), driver.Value(tccFenceDo.BranchId)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}

	err = GetTccFenceStoreDatabaseMapper().DeleteTCCFenceDO(tx, tccFenceDo.Xid, tccFenceDo.BranchId)
	tx.Commit()
	assert.Equal(t, nil, err)
}

func TestTccFenceStoreDatabaseMapper_DeleteTCCFenceDOByMdfDate(t *testing.T) {
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		GmtCreate: now,
	}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare(sql2.GetDeleteSQLByMdfDateAndStatus("tcc_fence_log")).
		ExpectExec().
		WithArgs(driver.Value(tccFenceDo.GmtModified.Add(math.MaxInt32))).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}
	err = GetTccFenceStoreDatabaseMapper().DeleteTCCFenceDOByMdfDate(tx, tccFenceDo.GmtModified.Add(math.MaxInt32))
	tx.Commit()
	assert.Equal(t, nil, err)
}
