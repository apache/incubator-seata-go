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

package undo

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type mockUndoLogManager struct {
	mock.Mock
}

func (m *mockUndoLogManager) Init() {
	m.Called()
}

func (m *mockUndoLogManager) DeleteUndoLog(ctx context.Context, xid string, branchID int64, conn *sql.Conn) error {
	args := m.Called(ctx, xid, branchID, conn)
	return args.Error(0)
}

func (m *mockUndoLogManager) BatchDeleteUndoLog(xid []string, branchID []int64, conn *sql.Conn) error {
	args := m.Called(xid, branchID, conn)
	return args.Error(0)
}

func (m *mockUndoLogManager) FlushUndoLog(tranCtx *types.TransactionContext, conn driver.Conn) error {
	args := m.Called(tranCtx, conn)
	return args.Error(0)
}

func (m *mockUndoLogManager) RunUndo(ctx context.Context, xid string, branchID int64, conn *sql.DB, dbName string) error {
	args := m.Called(ctx, xid, branchID, conn, dbName)
	return args.Error(0)
}

func (m *mockUndoLogManager) DBType() types.DBType {
	args := m.Called()
	return args.Get(0).(types.DBType)
}

func (m *mockUndoLogManager) HasUndoLogTable(ctx context.Context, conn *sql.Conn) (bool, error) {
	args := m.Called(ctx, conn)
	return args.Bool(0), args.Error(1)
}

type mockUndoLogBuilder struct {
	mock.Mock
}

func (m *mockUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	args := m.Called(ctx, execCtx)
	return args.Get(0).([]*types.RecordImage), args.Error(1)
}

func (m *mockUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	args := m.Called(ctx, execCtx, beforeImages)
	return args.Get(0).([]*types.RecordImage), args.Error(1)
}

func (m *mockUndoLogBuilder) GetExecutorType() types.ExecutorType {
	args := m.Called()
	return args.Get(0).(types.ExecutorType)
}

func TestRegisterUndoLogManager(t *testing.T) {
	tests := []struct {
		name    string
		manager UndoLogManager
		dbType  types.DBType
		wantErr bool
	}{
		{
			name:    "register mysql manager",
			manager: &mockUndoLogManager{},
			dbType:  types.DBTypeMySQL,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := tt.manager.(*mockUndoLogManager)
			mockMgr.On("DBType").Return(tt.dbType)
			mockMgr.On("Init").Return()

			err := RegisterUndoLogManager(tt.manager)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			err = RegisterUndoLogManager(tt.manager)
			assert.NoError(t, err)
		})
	}
}

func TestRegisterUndoLogBuilder(t *testing.T) {
	executorType := types.InsertExecutor
	builderFunc := func() UndoLogBuilder {
		return &mockUndoLogBuilder{}
	}

	RegisterUndoLogBuilder(executorType, builderFunc)

	builder := GetUndologBuilder(executorType)
	assert.NotNil(t, builder)

	RegisterUndoLogBuilder(executorType, builderFunc)
	builder2 := GetUndologBuilder(executorType)
	assert.NotNil(t, builder2)
}

func TestGetUndologBuilder(t *testing.T) {
	tests := []struct {
		name         string
		executorType types.ExecutorType
		setup        func()
		want         bool
	}{
		{
			name:         "get existing builder",
			executorType: types.UpdateExecutor,
			setup: func() {
				RegisterUndoLogBuilder(types.UpdateExecutor, func() UndoLogBuilder {
					return &mockUndoLogBuilder{}
				})
			},
			want: true,
		},
		{
			name:         "get non-existing builder",
			executorType: types.DeleteExecutor,
			setup:        func() {},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			builder := GetUndologBuilder(tt.executorType)
			if tt.want {
				assert.NotNil(t, builder)
			} else {
				assert.Nil(t, builder)
			}
		})
	}
}

func TestGetUndoLogManager(t *testing.T) {
	mockMgr := &mockUndoLogManager{}
	mockMgr.On("DBType").Return(types.DBTypeMySQL)
	mockMgr.On("Init").Return()

	err := RegisterUndoLogManager(mockMgr)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		dbType  types.DBType
		wantErr bool
	}{
		{
			name:    "get existing manager",
			dbType:  types.DBTypeMySQL,
			wantErr: false,
		},
		{
			name:    "get non-existing manager",
			dbType:  types.DBTypePostgreSQL,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := GetUndoLogManager(tt.dbType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
				assert.Contains(t, err.Error(), "not found UndoLogManager")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestUndologRecord_CanUndo(t *testing.T) {
	tests := []struct {
		name      string
		logStatus UndoLogStatue
		want      bool
	}{
		{
			name:      "can undo - normal status",
			logStatus: UndoLogStatueNormnal,
			want:      true,
		},
		{
			name:      "cannot undo - global finished",
			logStatus: UndoLogStatueGlobalFinished,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &UndologRecord{
				LogStatus: tt.logStatus,
			}
			assert.Equal(t, tt.want, record.CanUndo())
		})
	}
}

func TestBranchUndoLog_Marshal(t *testing.T) {
	branchLog := &BranchUndoLog{
		Xid:      "test-xid",
		BranchID: 123,
		Logs:     []SQLUndoLog{},
	}

	result := branchLog.Marshal()
	assert.Nil(t, result)
}

func TestBranchUndoLog_Reverse(t *testing.T) {
	tests := []struct {
		name     string
		logs     []SQLUndoLog
		expected []string
	}{
		{
			name: "empty logs",
			logs: []SQLUndoLog{},
		},
		{
			name: "single log",
			logs: []SQLUndoLog{
				{TableName: "table1"},
			},
			expected: []string{"table1"},
		},
		{
			name: "multiple logs",
			logs: []SQLUndoLog{
				{TableName: "table1"},
				{TableName: "table2"},
				{TableName: "table3"},
			},
			expected: []string{"table3", "table2", "table1"},
		},
		{
			name: "even number of logs",
			logs: []SQLUndoLog{
				{TableName: "table1"},
				{TableName: "table2"},
				{TableName: "table3"},
				{TableName: "table4"},
			},
			expected: []string{"table4", "table3", "table2", "table1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branchLog := &BranchUndoLog{
				Logs: tt.logs,
			}
			branchLog.Reverse()

			if len(tt.expected) == 0 {
				assert.Equal(t, 0, len(branchLog.Logs))
			} else {
				assert.Equal(t, len(tt.expected), len(branchLog.Logs))
				for i, expectedTable := range tt.expected {
					assert.Equal(t, expectedTable, branchLog.Logs[i].TableName)
				}
			}
		})
	}
}

func TestSQLUndoLog_SetTableMeta(t *testing.T) {
	tableMeta := &types.TableMeta{
		TableName: "test_table",
	}

	beforeImage := &types.RecordImage{}
	afterImage := &types.RecordImage{}

	tests := []struct {
		name        string
		beforeImage *types.RecordImage
		afterImage  *types.RecordImage
	}{
		{
			name:        "both images exist",
			beforeImage: beforeImage,
			afterImage:  afterImage,
		},
		{
			name:        "only before image",
			beforeImage: beforeImage,
			afterImage:  nil,
		},
		{
			name:        "only after image",
			beforeImage: nil,
			afterImage:  afterImage,
		},
		{
			name:        "no images",
			beforeImage: nil,
			afterImage:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlLog := SQLUndoLog{
				BeforeImage: tt.beforeImage,
				AfterImage:  tt.afterImage,
			}

			sqlLog.SetTableMeta(tableMeta)

			if tt.beforeImage != nil {
				assert.Equal(t, tableMeta, tt.beforeImage.TableMeta)
				assert.Equal(t, tableMeta.TableName, tt.beforeImage.TableName)
			}
			if tt.afterImage != nil {
				assert.Equal(t, tableMeta, tt.afterImage.TableMeta)
				assert.Equal(t, tableMeta.TableName, tt.afterImage.TableName)
			}
		})
	}
}
