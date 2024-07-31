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

package at

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestNewPlainExecutor(t *testing.T) {
	executor := NewPlainExecutor(nil, nil)
	_, ok := executor.(*plainExecutor)
	assert.Equalf(t, true, ok, "should be *plainExecutor")
}

func TestPlainExecutor_ExecContext(t *testing.T) {
	tests := []struct {
		name    string
		f       exec.CallbackWithNamedValue
		wantVal types.ExecResult
		wantErr error
	}{
		{
			name: "test1",
			f: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return NewMockInsertResult(int64(1), int64(2)), nil
			},
			wantVal: NewMockInsertResult(int64(1), int64(2)),
			wantErr: nil,
		},
		{
			name: "test2",
			f: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return nil, fmt.Errorf("test error")
			},
			wantVal: nil,
			wantErr: fmt.Errorf("test error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &plainExecutor{execContext: &types.ExecContext{}}
			val, err := u.ExecContext(context.Background(), tt.f)
			assert.Equalf(t, tt.wantVal, val, "")
			assert.Equalf(t, tt.wantErr, err, "")
		})
	}
}

type mockInsertResult struct {
	lastInsertID int64
	rowsAffected int64
}

func NewMockInsertResult(lastInsertID int64, rowsAffected int64) mockInsertResult {
	return mockInsertResult{
		lastInsertID: lastInsertID,
		rowsAffected: rowsAffected,
	}
}

func (m mockInsertResult) GetRows() driver.Rows {
	return &mock.MockTestDriverRows{}
}

func (m mockInsertResult) GetResult() driver.Result {
	return sqlmock.NewResult(m.lastInsertID, m.rowsAffected)
}
