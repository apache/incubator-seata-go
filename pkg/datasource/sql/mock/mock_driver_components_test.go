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

func TestNewMockTestDriverStmt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStmt := NewMockTestDriverStmt(ctrl)
	assert.NotNil(t, mockStmt)
	assert.NotNil(t, mockStmt.EXPECT())
}

func TestMockTestDriverStmt_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStmt := NewMockTestDriverStmt(ctrl)

	mockStmt.EXPECT().Close().Return(nil)

	err := mockStmt.Close()
	assert.NoError(t, err)
}

func TestMockTestDriverStmt_NumInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStmt := NewMockTestDriverStmt(ctrl)
	expectedNum := 2

	mockStmt.EXPECT().NumInput().Return(expectedNum)

	num := mockStmt.NumInput()
	assert.Equal(t, expectedNum, num)
}

func TestMockTestDriverStmt_Exec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStmt := NewMockTestDriverStmt(ctrl)
	args := []driver.Value{"test", 123}
	result := driver.Result(nil)

	mockStmt.EXPECT().Exec(args).Return(result, nil)

	res, err := mockStmt.Exec(args)
	assert.NoError(t, err)
	assert.Equal(t, result, res)
}

func TestMockTestDriverStmt_ExecContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStmt := NewMockTestDriverStmt(ctrl)
	ctx := context.Background()
	args := []driver.NamedValue{{Value: "test"}, {Value: 123}}
	result := driver.Result(nil)

	mockStmt.EXPECT().ExecContext(ctx, args).Return(result, nil)

	res, err := mockStmt.ExecContext(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, result, res)
}

func TestMockTestDriverStmt_Query(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStmt := NewMockTestDriverStmt(ctrl)
	mockRows := NewMockTestDriverRows(ctrl)
	args := []driver.Value{"test"}

	mockStmt.EXPECT().Query(args).Return(mockRows, nil)

	rows, err := mockStmt.Query(args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, rows)
}

func TestMockTestDriverStmt_QueryContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStmt := NewMockTestDriverStmt(ctrl)
	mockRows := NewMockTestDriverRows(ctrl)
	ctx := context.Background()
	args := []driver.NamedValue{{Value: "test"}}

	mockStmt.EXPECT().QueryContext(ctx, args).Return(mockRows, nil)

	rows, err := mockStmt.QueryContext(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, rows)
}

func TestNewMockTestDriverTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTx := NewMockTestDriverTx(ctrl)
	assert.NotNil(t, mockTx)
	assert.NotNil(t, mockTx.EXPECT())
}

func TestMockTestDriverTx_Commit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTx := NewMockTestDriverTx(ctrl)

	mockTx.EXPECT().Commit().Return(nil)

	err := mockTx.Commit()
	assert.NoError(t, err)
}

func TestMockTestDriverTx_Rollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTx := NewMockTestDriverTx(ctrl)

	mockTx.EXPECT().Rollback().Return(nil)

	err := mockTx.Rollback()
	assert.NoError(t, err)
}

func TestNewMockTestDriverRows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRows := NewMockTestDriverRows(ctrl)
	assert.NotNil(t, mockRows)
	assert.NotNil(t, mockRows.EXPECT())
}

func TestMockTestDriverRows_Columns(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRows := NewMockTestDriverRows(ctrl)
	expectedColumns := []string{"id", "name", "age"}

	mockRows.EXPECT().Columns().Return(expectedColumns)

	columns := mockRows.Columns()
	assert.Equal(t, expectedColumns, columns)
}

func TestMockTestDriverRows_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRows := NewMockTestDriverRows(ctrl)

	mockRows.EXPECT().Close().Return(nil)

	err := mockRows.Close()
	assert.NoError(t, err)
}

func TestMockTestDriverRows_Next(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRows := NewMockTestDriverRows(ctrl)
	dest := make([]driver.Value, 3)

	mockRows.EXPECT().Next(dest).Return(nil)

	err := mockRows.Next(dest)
	assert.NoError(t, err)
}
