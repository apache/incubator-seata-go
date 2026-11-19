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
	"errors"
	"fmt"
	"sync"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

var (
	atExecutorsMu sync.RWMutex
	atExecutors   = make(map[types.DBType]func() SQLExecutor)
	// ErrATExecutorBuilderNotRegistered indicates the db type has no AT executor root registered.
	ErrATExecutorBuilderNotRegistered = errors.New("at executor builder not registered")
)

// RegisterATExecutor sets the builder for dbType. AT resources capture the
// current builder when they are opened.
func RegisterATExecutor(dt types.DBType, builder func() SQLExecutor) {
	atExecutorsMu.Lock()
	atExecutors[dt] = builder
	atExecutorsMu.Unlock()
}

// GetATExecutorBuilder returns the current builder for dbType, or nil if none is registered.
func GetATExecutorBuilder(dbType types.DBType) ATExecutorBuilder {
	atExecutorsMu.RLock()
	defer atExecutorsMu.RUnlock()
	return atExecutors[dbType]
}

// ATExecutorBuilder creates a fresh AT root executor for each statement.
type ATExecutorBuilder func() SQLExecutor

type (
	CallbackWithNamedValue func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error)

	CallbackWithValue func(ctx context.Context, query string, args []driver.Value) (types.ExecResult, error)

	SQLExecutor interface {
		Interceptors(interceptors []SQLHook)
		ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error)
		ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error)
	}
)

// BuildExecutor use db type and transaction type to build an executor. the executor can
// add custom hook, and intercept the user's business sql to generate the undo log.
func BuildExecutor(dbType types.DBType, transactionMode types.TransactionMode, query string) (SQLExecutor, error) {
	var builder ATExecutorBuilder
	if transactionMode != types.XAMode {
		builder = GetATExecutorBuilder(dbType)
	}
	return BuildExecutorWithATBuilder(dbType, transactionMode, query, builder)
}

// BuildExecutorWithATBuilder builds an executor using the supplied builder when an AT root is needed.
func BuildExecutorWithATBuilder(dbType types.DBType, transactionMode types.TransactionMode, query string, builder ATExecutorBuilder) (SQLExecutor, error) {
	parsed, err := parser.ParseSQLForDB(query, dbType)
	return buildExecutorWithParsedSQL(dbType, transactionMode, builder, parsed, err)
}

// ExecuteATWithBuilder builds a per-execution root from a resource's bound
// builder. MySQL AT can reuse the AST used for hook selection in this call.
func ExecuteATWithBuilder(ctx context.Context, dbType types.DBType, execCtx *types.ExecContext, builder ATExecutorBuilder, run CallbackWithNamedValue) (types.ExecResult, error) {
	query := execCtx.Query
	mode := execCtx.TxCtx.TransactionMode
	parsed, parseErr := parser.ParseSQLForDB(query, dbType)
	root, err := buildExecutorWithParsedSQL(dbType, mode, builder, parsed, parseErr)
	if err != nil {
		return nil, err
	}
	if dbType == types.DBTypeMySQL && parsed != nil && mode != types.XAMode &&
		(mode == types.ATMode || execCtx.IsRequireGlobalLock) {
		restoreParse := execCtx.BindATParse(query, parsed)
		defer restoreParse()
	}
	return root.ExecWithNamedValue(ctx, execCtx, run)
}

// Hook selection only needs the statement type. The root binds table
// identities when it needs metadata or images.
func buildExecutorWithParsedSQL(dbType types.DBType, transactionMode types.TransactionMode, builder ATExecutorBuilder, parseContext *types.ParseContext, err error) (SQLExecutor, error) {
	if err != nil {
		// XA mode uses a pass-through executor and never needs the parsed statement:
		// two-phase commit is managed by the TC and no undo log is generated, so SQL
		// the shared parser cannot understand must still execute as-is instead of
		// erroring or falling back to an AT executor. This check must come before the
		// PostgreSQL fallback so XA mode is never routed to an AT executor.
		if transactionMode == types.XAMode {
			return newXAExecutor(commonHook), nil
		}
		if transactionMode == types.Local {
			// The executor has the context needed to decide whether global-lock
			// handling is required. Ordinary local SQL can execute unchanged.
			return newATExecutor(dbType, builder, append([]SQLHook{}, commonHook...))
		}
		if dbType == types.DBTypePostgreSQL {
			// PostgreSQL local execution must remain pass-through even when the
			// shared parser cannot understand PostgreSQL-only syntax yet.
			return newATExecutor(dbType, builder, nil)
		}
		return nil, err
	}

	hooks := make([]SQLHook, 0, 4)
	hooks = append(hooks, commonHook...)
	hooks = append(hooks, hookSolts[parseContext.SQLType]...)

	// For XA mode, use a plain executor without AT-specific hooks: XA transactions
	// don't need undo logs, they rely on the two-phase commit protocol managed by TC.
	if transactionMode == types.XAMode {
		return newXAExecutor(hooks), nil
	}

	return newATExecutor(dbType, builder, hooks)
}

// newXAExecutor builds a pass-through executor for XA mode. It applies the given
// hooks but never wraps an AT executor, so no undo log is generated.
func newXAExecutor(hooks []SQLHook) SQLExecutor {
	e := &BaseExecutor{}
	e.Interceptors(hooks)
	return e
}

func newATExecutor(dbType types.DBType, builder ATExecutorBuilder, hooks []SQLHook) (SQLExecutor, error) {
	if builder == nil {
		return nil, fmt.Errorf("%w: dbType=%d", ErrATExecutorBuilderNotRegistered, dbType)
	}

	e := builder()
	if e == nil {
		return nil, fmt.Errorf("%w: dbType=%d", ErrATExecutorBuilderNotRegistered, dbType)
	}
	e.Interceptors(hooks)
	return e, nil
}

type BaseExecutor struct {
	hooks []SQLHook
	ex    SQLExecutor
}

func (e *BaseExecutor) Interceptors(interceptors []SQLHook) {
	e.hooks = interceptors
}

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
func (e *BaseExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
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

	nvargs := make([]driver.NamedValue, 0, len(execCtx.Values))
	for i, value := range execCtx.Values {
		nvargs = append(nvargs, driver.NamedValue{
			Value:   value,
			Ordinal: i,
		})
	}

	return f(ctx, execCtx.Query, nvargs)
}
