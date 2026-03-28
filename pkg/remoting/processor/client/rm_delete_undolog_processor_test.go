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

package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
)

func TestProcess_InvalidBodyType(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	err := p.Process(context.Background(), message.RpcMessage{
		Body: "not a UndoLogDeleteRequest",
	})
	assert.Error(t, err)
}

func TestProcess_NonATBranchType_Skipped(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	for _, bt := range []branch.BranchType{branch.BranchTypeXA, branch.BranchTypeTCC, branch.BranchTypeSAGA} {
		err := p.Process(context.Background(), message.RpcMessage{
			Body: message.UndoLogDeleteRequest{
				ResourceId: "any-resource",
				SaveDays:   7,
				BranchType: bt,
			},
		})
		assert.NoError(t, err, "branch type %v should be skipped silently", bt)
	}
}

func TestProcess_ResourceNotFound_ReturnsNil(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	err := p.Process(context.Background(), message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: "jdbc:mysql://not-registered:3306/db",
			SaveDays:   7,
			BranchType: branch.BranchTypeAT,
		},
	})
	// GetResourceManager panic 和 resource 不在缓存，都会被 deleteExpiredUndoLog
	// 的调用方吞掉记日志，Process 必须返回 nil
	assert.NoError(t, err, "TC one-way: all internal errors must be swallowed")
}

func TestProcess_NegativeSaveDays_ReturnsNil(t *testing.T) {
	p := &rmDeleteUndoLogProcessor{}
	err := p.Process(context.Background(), message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: "jdbc:mysql://not-registered:3306/db",
			SaveDays:   -32768,
			BranchType: branch.BranchTypeAT,
		},
	})
	assert.NoError(t, err)
}

type mockDBResource struct {
	db     *sql.DB
	dbType types.DBType
}

func (m *mockDBResource) GetDB() *sql.DB          { return m.db }
func (m *mockDBResource) GetDbType() types.DBType { return m.dbType }

func TestBatchDeleteByLogCreated_DeletesRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	before := time.Now().AddDate(0, 0, -7)

	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1000))
	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 50))

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, before)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchDeleteByLogCreated_CustomTableName(t *testing.T) {
	original := undo.UndoConfig.LogTable
	undo.UndoConfig.LogTable = "my_undo_log"
	defer func() { undo.UndoConfig.LogTable = original }()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("DELETE FROM my_undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, time.Now())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchDeleteByLogCreated_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("DELETE FROM undo_log WHERE log_created <= ?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(driver.ErrBadConn)

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	p := &rmDeleteUndoLogProcessor{}
	err = p.batchDeleteByLogCreated(context.Background(), conn, time.Now())
	assert.Error(t, err)
}

func BenchmarkProcess_NonAT(b *testing.B) {
	p := &rmDeleteUndoLogProcessor{}
	msg := message.RpcMessage{
		Body: message.UndoLogDeleteRequest{
			ResourceId: "any",
			SaveDays:   7,
			BranchType: branch.BranchTypeXA,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Process(context.Background(), msg)
	}
}
