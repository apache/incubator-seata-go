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

	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/util/log"
)

// executorSolts
var executorSolts = make(map[types.DBType]map[parser.ExecutorType]func() SQLExecutor)

func RegisterExecutor(dt types.DBType, et parser.ExecutorType, builder func() SQLExecutor) {
	if _, ok := executorSolts[dt]; !ok {
		executorSolts[dt] = make(map[parser.ExecutorType]func() SQLExecutor)
	}

	val := executorSolts[dt]

	val[et] = func() SQLExecutor {
		return &BaseExecutor{ex: builder()}
	}
}

type (
	CallbackWithNamedValue func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error)

	CallbackWithValue func(ctx context.Context, query string, args []driver.Value) (types.ExecResult, error)

	SQLExecutor interface {
		// Interceptors
		interceptors(interceptors []SQLHook)
		// Exec
		ExecWithNamedValue(ctx context.Context, execCtx *ExecContext, f CallbackWithNamedValue) (types.ExecResult, error)
		// Exec
		ExecWithValue(ctx context.Context, execCtx *ExecContext, f CallbackWithValue) (types.ExecResult, error)
	}
)

// BuildExecutor
func BuildExecutor(dbType types.DBType, txType types.TransactionType, query string) (SQLExecutor, error) {
	if txType == types.XAMode {
		hooks := make([]SQLHook, 0, 4)
		hooks = append(hooks, commonHook...)

		e := &BaseExecutor{}
		e.interceptors(hooks)
		return e, nil
	}

	parseCtx, err := parser.DoParser(query)
	if err != nil {
		return nil, err
	}

	hooks := make([]SQLHook, 0, 4)
	hooks = append(hooks, commonHook...)
	hooks = append(hooks, hookSolts[parseCtx.SQLType]...)

	factories, ok := executorSolts[dbType]
	if !ok {
		log.Debugf("%s not found executor factories, return default Executor", dbType.String())
		e := &BaseExecutor{}
		e.interceptors(hooks)
		return e, nil
	}

	supplier, ok := factories[parseCtx.ExecutorType]
	if !ok {
		log.Debugf("%s not found executor for %s, return default Executor",
			dbType.String(), parseCtx.ExecutorType.String())
		e := &BaseExecutor{}
		e.interceptors(hooks)
		return e, nil
	}

	executor := supplier()
	executor.interceptors(hooks)
	return executor, nil
}

type BaseExecutor struct {
	is []SQLHook
	ex SQLExecutor
}

// Interceptors
func (e *BaseExecutor) interceptors(interceptors []SQLHook) {
	e.is = interceptors
}

// ExecWithNamedValue
func (e *BaseExecutor) ExecWithNamedValue(ctx context.Context, execCtx *ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
	for i := range e.is {
		e.is[i].Before(ctx, execCtx)
	}

	defer func() {
		for i := range e.is {
			e.is[i].After(ctx, execCtx)
		}
	}()

	if e.ex != nil {
		return e.ex.ExecWithNamedValue(ctx, execCtx, f)
	}

	return f(ctx, execCtx.Query, execCtx.NamedValues)
}

// ExecWithValue
func (e *BaseExecutor) ExecWithValue(ctx context.Context, execCtx *ExecContext, f CallbackWithValue) (types.ExecResult, error) {
	for i := range e.is {
		e.is[i].Before(ctx, execCtx)
	}

	defer func() {
		for i := range e.is {
			e.is[i].After(ctx, execCtx)
		}
	}()

	if e.ex != nil {
		return e.ex.ExecWithValue(ctx, execCtx, f)
	}

	return f(ctx, execCtx.Query, execCtx.Values)
}
