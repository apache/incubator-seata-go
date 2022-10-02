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

package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/mock/gomock"
	"github.com/seata/seata-go/pkg/datasource/sql/mock"
	"github.com/seata/seata-go/pkg/util/log"
)

var db *sql.DB

func Test_SQLOpen(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(t, ctrl)
	_ = mockMgr

	log.Info("begin test")
	var err error
	db, err = sql.Open("seata-at-mysql", "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Commit().AnyTimes().Return(nil)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockStmt := mock.NewMockTestDriverStmt(ctrl)
		mockStmt.EXPECT().ExecContext(gomock.Any(), gomock.Any()).AnyTimes().Return(driver.ResultNoRows, nil)
		mockStmt.EXPECT().Exec(gomock.Any()).AnyTimes().Return(driver.ResultNoRows, nil)
		mockStmt.EXPECT().Close().AnyTimes().Return(nil)
		mockStmt.EXPECT().NumInput().AnyTimes().Return(2)

		mockRows := mock.NewMockTestDriverRows(ctrl)
		mockRows.EXPECT().Close().AnyTimes().Return(nil)
		mockRows.EXPECT().Columns().AnyTimes().Return([]string{"id", "name"})
		mockRows.EXPECT().Next(gomock.Any()).AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().Prepare(gomock.Any()).AnyTimes().Return(mockStmt, nil)
		mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).AnyTimes().Return(mockStmt, nil)
		mockConn.EXPECT().Query(gomock.Any(), gomock.Any()).AnyTimes().Return(mockRows, nil)
		mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(mockRows, nil)
		baseMoclConn(mockConn)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	wait := sync.WaitGroup{}

	txInvoker := func(prefix string, offset, total int) {
		defer wait.Done()

		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			t.Fatal(err)
		}

		stmt, err := tx.Prepare("insert into foo(id, name) values(?, ?)")
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()
		for i := 0; i < total; i++ {
			_, err = stmt.Exec(i+offset, fmt.Sprintf("%s-%03d", prefix, i))
			if err != nil {
				t.Fatal(err)
			}
		}
		err = tx.Commit()
		if err != nil {
			t.Fatal(err)
		}
	}

	wait.Add(2)

	t.Parallel()
	t.Run("", func(t *testing.T) {
		txInvoker("seata-go-at-1", 0, 10)
	})
	t.Run("", func(t *testing.T) {
		txInvoker("seata-go-at-2", 20, 10)
	})

	wait.Wait()
	queryMultiRow()
}

func queryMultiRow() {
	sqlStr := "select id, name from foo where id > ?"
	rows, err := db.Query(sqlStr, 0)
	if err != nil {
		fmt.Printf("query failed, err:%v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var u user
		err := rows.Scan(&u.id, &u.name)
		if err != nil {
			fmt.Printf("scan failed, err:%v\n", err)
			return
		}
		fmt.Printf("id:%d username:%s password:%s\n", u.id, u.name, u.name)
	}
}

type user struct {
	id   int
	name string
}
