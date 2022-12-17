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
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/builder"
	"github.com/seata/seata-go/pkg/util/log"
)

func init() {
	undo.RegisterUndoLogBuilder(types.UpdateExecutor, builder.GetMySQLUpdateUndoLogBuilder)
	undo.RegisterUndoLogBuilder(types.MultiExecutor, builder.GetMySQLMultiUndoLogBuilder)
}

var (
	executorSoltsAT = make(map[types.DBType]map[types.ExecutorType]func() SQLExecutor)
	executorSoltsXA = make(map[types.DBType]func() SQLExecutor)
)

// RegisterATExecutor AT executor
func RegisterATExecutor(dt types.DBType, et types.ExecutorType, builder func() SQLExecutor) {
	if _, ok := executorSoltsAT[dt]; !ok {
		executorSoltsAT[dt] = make(map[types.ExecutorType]func() SQLExecutor)
	}

	val := executorSoltsAT[dt]

	val[et] = func() SQLExecutor {
		return &BaseExecutor{ex: builder()}
	}
}

// RegisterXAExecutor XA executor
func RegisterXAExecutor(dt types.DBType, builder func() SQLExecutor) {
	executorSoltsXA[dt] = func() SQLExecutor {
		return builder()
	}
}

type (
	CallbackWithNamedValue func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error)

	CallbackWithValue func(ctx context.Context, query string, args []driver.Value) (types.ExecResult, error)

	SQLExecutor interface {
		Interceptors(interceptors []SQLHook)
		ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error)
		ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithValue) (types.ExecResult, error)
	}
)

// BuildExecutor use db type and transaction type to build an executor. the executor can
// add custom hook, and intercept the user's business sql to generate the undo log.
func BuildExecutor(dbType types.DBType, transactionMode types.TransactionMode, query string) (SQLExecutor, error) {
	parseContext, err := parser.DoParser(query)
	if err != nil {
		return nil, err
	}

	hooks := make([]SQLHook, 0, 4)
	hooks = append(hooks, commonHook...)
	hooks = append(hooks, hookSolts[parseContext.SQLType]...)

	if transactionMode == types.XAMode {
		e := executorSoltsXA[dbType]()
		e.Interceptors(hooks)
		return e, nil
	}

	if transactionMode == types.ATMode {
		e := executorSoltsAT[dbType][parseContext.ExecutorType]()
		e.Interceptors(hooks)
		return e, nil
	}

	factories, ok := executorSoltsAT[dbType]
	if !ok {
		log.Debugf("%s not found executor factories, return default Executor", dbType.String())
		e := &BaseExecutor{}
		e.Interceptors(hooks)
		return e, nil
	}

	supplier, ok := factories[parseContext.ExecutorType]
	if !ok {
		log.Debugf("%s not found executor for %s, return default Executor",
			dbType.String(), parseContext.ExecutorType)
		e := &BaseExecutor{}
		e.Interceptors(hooks)
		return e, nil
	}

	executor := supplier()
	executor.Interceptors(hooks)
	return executor, nil
}

type BaseExecutor struct {
	hooks []SQLHook
	ex    SQLExecutor
}

// Interceptors
func (e *BaseExecutor) Interceptors(interceptors []SQLHook) {
	e.hooks = interceptors
}

// ExecWithNamedValue
func (e *BaseExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
	for i := range e.hooks {
		e.hooks[i].Before(ctx, execCtx)
	}

	defer func() {
		for i := range e.hooks {
			e.hooks[i].After(ctx, execCtx)
		}
	}()

	if e.ex != nil {
		return e.ex.ExecWithNamedValue(ctx, execCtx, f)
	}

	return f(ctx, execCtx.Query, execCtx.NamedValues)
}

// ExecWithValue
func (e *BaseExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithValue) (types.ExecResult, error) {
	for i := range e.hooks {
		e.hooks[i].Before(ctx, execCtx)
	}

	defer func() {
		for i := range e.hooks {
			e.hooks[i].After(ctx, execCtx)
		}
	}()

	if e.ex != nil {
		return e.ex.ExecWithValue(ctx, execCtx, f)
	}

	return f(ctx, execCtx.Query, execCtx.Values)
}
