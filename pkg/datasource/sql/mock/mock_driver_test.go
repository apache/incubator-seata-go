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

package mock

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewMockTestDriver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDriver := NewMockTestDriver(ctrl)
	assert.NotNil(t, mockDriver)
	assert.NotNil(t, mockDriver.EXPECT())
}

func TestMockTestDriver_Open(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDriver := NewMockTestDriver(ctrl)
	mockConn := NewMockTestDriverConn(ctrl)
	dsn := "test:test@tcp(localhost:3306)/test"

	mockDriver.EXPECT().Open(dsn).Return(mockConn, nil)

	conn, err := mockDriver.Open(dsn)
	assert.NoError(t, err)
	assert.Equal(t, mockConn, conn)
}

func TestNewMockTestDriverConnector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnector := NewMockTestDriverConnector(ctrl)
	assert.NotNil(t, mockConnector)
	assert.NotNil(t, mockConnector.EXPECT())
}

func TestMockTestDriverConnector_Connect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnector := NewMockTestDriverConnector(ctrl)
	mockConn := NewMockTestDriverConn(ctrl)
	ctx := context.Background()

	mockConnector.EXPECT().Connect(ctx).Return(mockConn, nil)

	conn, err := mockConnector.Connect(ctx)
	assert.NoError(t, err)
	assert.Equal(t, mockConn, conn)
}

func TestMockTestDriverConnector_Driver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnector := NewMockTestDriverConnector(ctrl)
	mockDriver := NewMockTestDriver(ctrl)

	mockConnector.EXPECT().Driver().Return(mockDriver)

	drv := mockConnector.Driver()
	assert.Equal(t, mockDriver, drv)
}

func TestNewMockTestDriverConn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	assert.NotNil(t, mockConn)
	assert.NotNil(t, mockConn.EXPECT())
}

func TestMockTestDriverConn_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)

	mockConn.EXPECT().Close().Return(nil)

	err := mockConn.Close()
	assert.NoError(t, err)
}

func TestMockTestDriverConn_Begin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	mockTx := NewMockTestDriverTx(ctrl)

	mockConn.EXPECT().Begin().Return(mockTx, nil)

	tx, err := mockConn.Begin()
	assert.NoError(t, err)
	assert.Equal(t, mockTx, tx)
}

func TestMockTestDriverConn_BeginTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	mockTx := NewMockTestDriverTx(ctrl)
	ctx := context.Background()
	opts := driver.TxOptions{}

	mockConn.EXPECT().BeginTx(ctx, opts).Return(mockTx, nil)

	tx, err := mockConn.BeginTx(ctx, opts)
	assert.NoError(t, err)
	assert.Equal(t, mockTx, tx)
}

func TestMockTestDriverConn_Prepare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	mockStmt := NewMockTestDriverStmt(ctrl)
	query := "SELECT * FROM test WHERE id = ?"

	mockConn.EXPECT().Prepare(query).Return(mockStmt, nil)

	stmt, err := mockConn.Prepare(query)
	assert.NoError(t, err)
	assert.Equal(t, mockStmt, stmt)
}

func TestMockTestDriverConn_PrepareContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	mockStmt := NewMockTestDriverStmt(ctrl)
	ctx := context.Background()
	query := "SELECT * FROM test WHERE id = ?"

	mockConn.EXPECT().PrepareContext(ctx, query).Return(mockStmt, nil)

	stmt, err := mockConn.PrepareContext(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, mockStmt, stmt)
}

func TestMockTestDriverConn_Exec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	query := "INSERT INTO test (name) VALUES (?)"
	args := []driver.Value{"test"}
	result := driver.Result(nil)

	mockConn.EXPECT().Exec(query, args).Return(result, nil)

	res, err := mockConn.Exec(query, args)
	assert.NoError(t, err)
	assert.Equal(t, result, res)
}

func TestMockTestDriverConn_ExecContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	ctx := context.Background()
	query := "INSERT INTO test (name) VALUES (?)"
	args := []driver.NamedValue{{Value: "test"}}
	result := driver.Result(nil)

	mockConn.EXPECT().ExecContext(ctx, query, args).Return(result, nil)

	res, err := mockConn.ExecContext(ctx, query, args)
	assert.NoError(t, err)
	assert.Equal(t, result, res)
}

func TestMockTestDriverConn_Query(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	mockRows := NewMockTestDriverRows(ctrl)
	query := "SELECT * FROM test"
	args := []driver.Value{}

	mockConn.EXPECT().Query(query, args).Return(mockRows, nil)

	rows, err := mockConn.Query(query, args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, rows)
}

func TestMockTestDriverConn_QueryContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	mockRows := NewMockTestDriverRows(ctrl)
	ctx := context.Background()
	query := "SELECT * FROM test"
	args := []driver.NamedValue{}

	mockConn.EXPECT().QueryContext(ctx, query, args).Return(mockRows, nil)

	rows, err := mockConn.QueryContext(ctx, query, args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, rows)
}

func TestMockTestDriverConn_Ping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	ctx := context.Background()

	mockConn.EXPECT().Ping(ctx).Return(nil)

	err := mockConn.Ping(ctx)
	assert.NoError(t, err)
}

func TestMockTestDriverConn_ResetSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := NewMockTestDriverConn(ctrl)
	ctx := context.Background()

	mockConn.EXPECT().ResetSession(ctx).Return(nil)

	err := mockConn.ResetSession(ctx)
	assert.NoError(t, err)
}
