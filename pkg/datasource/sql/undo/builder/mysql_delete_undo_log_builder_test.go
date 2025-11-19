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

package builder

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/util/log"
)

func TestBuildDeleteBeforeImageSQL(t *testing.T) {
	log.Init()
	var (
		builder = MySQLDeleteUndoLogBuilder{}
	)
	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "delete from t_user where id = ?",
			sourceQueryArgs: []driver.Value{100},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
		{
			sourceQuery:     "delete from t_user where id = ? and name = 'Jack' and age between ? and ?",
			sourceQueryArgs: []driver.Value{100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "delete from t_user where id = ? and name = 'Jack' and age in (?,?)",
			sourceQueryArgs: []driver.Value{100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "delete from t_user where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc limit ?",
			sourceQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE kk BETWEEN ? AND ? AND id=? AND addr IN (?,?) AND age>? ORDER BY name DESC LIMIT ? FOR UPDATE",
			expectQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := builder.buildBeforeImageSQL(tt.sourceQuery, tt.sourceQueryArgs)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, args)
		})
	}
}

func TestGetMySQLDeleteUndoLogBuilder(t *testing.T) {
	builder := GetMySQLDeleteUndoLogBuilder()
	assert.NotNil(t, builder)
	assert.IsType(t, &MySQLDeleteUndoLogBuilder{}, builder)
}

func TestMySQLDeleteUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := &MySQLDeleteUndoLogBuilder{}
	executorType := builder.GetExecutorType()
	assert.Equal(t, types.DeleteExecutor, executorType)
}

func TestMySQLDeleteUndoLogBuilder_BeforeImage(t *testing.T) {
	log.Init()
	builder := &MySQLDeleteUndoLogBuilder{}

	// Test with nil values - should handle NamedValues
	execCtx := &types.ExecContext{
		Values: nil,
		NamedValues: []driver.NamedValue{
			{Name: "0", Value: 100},
		},
		Query: "DELETE FROM t_user WHERE id = ?",
		Conn:  nil,
	}

	// Since conn is nil, this should panic or return error, let's catch the panic
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil conn or index error
			panicStr := fmt.Sprintf("%v", r)
			assert.True(t,
				strings.Contains(panicStr, "nil pointer") ||
					strings.Contains(panicStr, "index out of range"),
				"Expected panic related to nil pointer or index error, got: %s", panicStr)
		}
	}()

	_, err := builder.BeforeImage(context.Background(), execCtx)
	// If we reach here, there should be an error
	if err == nil {
		t.Error("Expected error or panic when conn is nil")
	}
}

func TestMySQLDeleteUndoLogBuilder_AfterImage(t *testing.T) {
	builder := &MySQLDeleteUndoLogBuilder{}

	execCtx := &types.ExecContext{}
	beforeImages := []*types.RecordImage{}

	images, err := builder.AfterImage(context.Background(), execCtx, beforeImages)
	// AfterImage for DELETE should return nil
	assert.NoError(t, err)
	assert.Nil(t, images)
}
