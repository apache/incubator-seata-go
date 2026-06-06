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

package xa

import (
	"context"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
)

type postgresMockRows struct {
	idx  int
	data [][]interface{}
}

func (m *postgresMockRows) Columns() []string { return []string{"gid"} }

func (m *postgresMockRows) Close() error { return nil }

func (m *postgresMockRows) Next(dest []driver.Value) error {
	if m.idx == len(m.data) {
		return io.EOF
	}

	for i := 0; i < len(dest) && i < len(m.data[m.idx]); i++ {
		dest[i] = m.data[m.idx][i]
	}
	m.idx++
	return nil
}

func TestPostgresXAConn_StartAndPrepare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockTx := mock.NewMockTestDriverTx(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
	mockConn.EXPECT().ExecContext(gomock.Any(), "PREPARE TRANSACTION 'xid'", gomock.Any()).Return(&driver.ResultNoRows, nil)

	conn := &PostgresXAConn{Conn: mockConn}
	assert.NoError(t, conn.Start(context.Background(), "xid", TMNoFlags))
	assert.NoError(t, conn.XAPrepare(context.Background(), "xid"))
	assert.Nil(t, conn.tx)
}

func TestPostgresXAConn_CommitPrepared(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), "COMMIT PREPARED 'xid'", gomock.Any()).Return(&driver.ResultNoRows, nil)

	conn := &PostgresXAConn{Conn: mockConn}
	assert.NoError(t, conn.Commit(context.Background(), "xid", false))
}

func TestPostgresXAConn_OnePhaseCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockTx := mock.NewMockTestDriverTx(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
	mockTx.EXPECT().Commit().Return(nil)

	conn := &PostgresXAConn{Conn: mockConn}
	assert.NoError(t, conn.Start(context.Background(), "xid", TMNoFlags))
	assert.NoError(t, conn.Commit(context.Background(), "xid", true))
	assert.Nil(t, conn.tx)
}

func TestPostgresXAConn_RollbackActiveTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockTx := mock.NewMockTestDriverTx(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
	mockTx.EXPECT().Rollback().Return(nil)

	conn := &PostgresXAConn{Conn: mockConn}
	assert.NoError(t, conn.Start(context.Background(), "xid", TMNoFlags))
	assert.NoError(t, conn.Rollback(context.Background(), "xid"))
	assert.Nil(t, conn.tx)
}

func TestPostgresXAConn_RollbackPrepared(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), "ROLLBACK PREPARED 'xid'", gomock.Any()).Return(&driver.ResultNoRows, nil)

	conn := &PostgresXAConn{Conn: mockConn}
	assert.NoError(t, conn.Rollback(context.Background(), "xid"))
}

func TestPostgresXAConn_Recover(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().QueryContext(gomock.Any(), "SELECT gid FROM pg_prepared_xacts WHERE database = current_database()", gomock.Any()).
		Return(&postgresMockRows{data: [][]interface{}{{"xid"}, {"another-xid"}}}, nil)

	conn := &PostgresXAConn{Conn: mockConn}
	got, err := conn.Recover(context.Background(), TMStartRScan|TMEndRScan)
	assert.NoError(t, err)
	assert.Equal(t, []string{"xid", "another-xid"}, got)
}
