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

package exec

import (
	"context"
	"database/sql/driver"

	"github.com/seata/seata-go-datasource/sql/parser"
	"github.com/seata/seata-go-datasource/sql/types"
)

var (
	// executorSolts
	executorSolts = make(map[types.DBType]map[parser.ExecutorType]func() SQLExecutor)
)

type (
	CallbackWithNamedValue func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error)

	CallbackWithValue func(ctx context.Context, query string, args []driver.Value) (types.ExecResult, error)

	SQLExecutor interface {
		// Interceptors
		interceptors(interceptors []SQLInterceptor)
		// Exec
		ExecWithNamedValue(tx *types.TransactionContext, ctx context.Context, query string, args []driver.NamedValue, f CallbackWithNamedValue) (types.ExecResult, error)
		// Exec
		ExecWithValue(tx *types.TransactionContext, ctx context.Context, query string, args []driver.Value, f CallbackWithValue) (types.ExecResult, error)
	}
)

// buildExecutor
func BuildExecutor(dbType types.DBType, query string) (SQLExecutor, error) {
	parseCtx, err := parser.DoParser(query)

	if err != nil {
		return nil, err
	}

	hooks := hookSolts[parseCtx.SQLType]

	executor := executorSolts[dbType][parseCtx.ExecutorType]()
	executor.interceptors(hooks)
	return executor, nil
}

type BaseExecutor struct {
}
