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
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"

	_ "github.com/go-sql-driver/mysql"
)

func TestTccFenceStoreDatabaseMapper_SetLogTableName(t *testing.T) {
	GetTccFenceStoreDatabaseMapperSingleton().SetLogTableName("tcc_fence_log")
	value := reflect.ValueOf(GetTccFenceStoreDatabaseMapperSingleton())
	assert.Equal(t, "tcc_fence_log", value.Elem().FieldByName("logTableName").String())
}

func TestTccFenceStoreDatabaseMapper_InsertTCCFenceDO(t *testing.T) {

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/seata")
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}
	defer conn.Close()
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		Xid:         "123123124124",
		BranchId:    12312312312,
		ActionName:  "fence_test",
		Status:      constant.StatusTried,
		GmtCreate:   now,
		GmtModified: now,
	}
	assert.Equal(t, true, GetTccFenceStoreDatabaseMapperSingleton().InsertTCCFenceDO(conn, tccFenceDo))
}

func TestTccFenceStoreDatabaseMapper_QueryTCCFenceDO(t *testing.T) {

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/seata?parseTime=true")
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}
	defer conn.Close()
	tccFenceDo := &model.TCCFenceDO{
		Xid:        "123123124124",
		BranchId:   12312312312,
		ActionName: "fence_test",
		Status:     constant.StatusTried,
	}
	actualFenceDo := GetTccFenceStoreDatabaseMapperSingleton().QueryTCCFenceDO(conn, tccFenceDo.Xid, tccFenceDo.BranchId)
	assert.Equal(t, tccFenceDo.Xid, actualFenceDo.Xid)
	assert.Equal(t, tccFenceDo.BranchId, actualFenceDo.BranchId)
	assert.Equal(t, tccFenceDo.Status, actualFenceDo.Status)
	assert.Equal(t, tccFenceDo.ActionName, actualFenceDo.ActionName)
	assert.NotEmpty(t, actualFenceDo.GmtModified)
	assert.NotEmpty(t, actualFenceDo.GmtCreate)
}

func TestTccFenceStoreDatabaseMapper_UpdateTCCFenceDO(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/seata")
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}
	defer conn.Close()
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		Xid:         "123123124124",
		BranchId:    12312312312,
		ActionName:  "fence_test",
		Status:      constant.StatusTried,
		GmtCreate:   now,
		GmtModified: now,
	}
	assert.Equal(t, true, GetTccFenceStoreDatabaseMapperSingleton().
		UpdateTCCFenceDO(conn, tccFenceDo.Xid, tccFenceDo.BranchId, tccFenceDo.Status, constant.StatusCommitted))
}

func TestTccFenceStoreDatabaseMapper_DeleteTCCFenceDO(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/seata")
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}
	defer conn.Close()
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		Xid:         "123123124124",
		BranchId:    12312312312,
		ActionName:  "fence_test",
		Status:      constant.StatusTried,
		GmtCreate:   now,
		GmtModified: now,
	}
	assert.Equal(t, true, GetTccFenceStoreDatabaseMapperSingleton().DeleteTCCFenceDO(conn, tccFenceDo.Xid, tccFenceDo.BranchId))
}

func TestTccFenceStoreDatabaseMapper_DeleteTCCFenceDOByMdfDate(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/seata")
	if err != nil {
		t.Fatalf("open db failed msg: %v", err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open conn failed msg :%v", err)
	}
	defer conn.Close()
	now := time.Now()
	tccFenceDo := &model.TCCFenceDO{
		Xid:         "123123124124",
		BranchId:    12312312312,
		ActionName:  "fence_test",
		Status:      constant.StatusCommitted,
		GmtCreate:   now,
		GmtModified: now,
	}
	assert.Equal(t, true, GetTccFenceStoreDatabaseMapperSingleton().InsertTCCFenceDO(conn, tccFenceDo))
	assert.Equal(t, true, GetTccFenceStoreDatabaseMapperSingleton().DeleteTCCFenceDOByMdfDate(conn, tccFenceDo.GmtModified.Add(math.MaxInt32)))
}
