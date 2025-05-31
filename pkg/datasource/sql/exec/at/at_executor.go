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

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/internal"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/mysql"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

func Init() {
	exec.RegisterATExecutor(types.DBTypeMySQL, func() exec.SQLExecutor { return &ATExecutor{} })
}

type ATExecutor struct {
	hooks []exec.SQLHook
}

func (e *ATExecutor) Interceptors(hooks []exec.SQLHook) {
	e.hooks = hooks
}

// ExecWithNamedValue find the executor by sql type
func (e *ATExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	queryParser, err := parser.DoParser(execCtx.Query)
	if err != nil {
		return nil, err
	}

	var executor internal.Executor

	if !tm.IsGlobalTx(ctx) {
		executor = internal.NewPlainExecutor(queryParser, execCtx)
	} else {
		switch queryParser.SQLType {
		case types.SQLTypeInsert:
			executor = e.NewInsertExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeUpdate:
			executor = e.NewUpdateExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeDelete:
			executor = e.NewDeleteExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeSelectForUpdate:
			executor = e.NewSelectForUpdateExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeInsertOnDuplicateUpdate:
			executor = e.NewInsertOnUpdateExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeMulti:
			executor = e.NewMultiExecutor(queryParser, execCtx, e.hooks)
		default:
			executor = internal.NewPlainExecutor(queryParser, execCtx)
		}
	}

	return executor.ExecContext(ctx, f)
}

// ExecWithValue transfer value to nameValue execute
func (e *ATExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	execCtx.NamedValues = util.ValueToNamedValue(execCtx.Values)
	return e.ExecWithNamedValue(ctx, execCtx, f)
}

func (e *ATExecutor) NewInsertExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	switch execContext.DBType {
	case types.DBTypeMySQL:
		return mysql.NewInsertExecutor(parserCtx, execContext, e.hooks)
	}
	log.Errorf("unsupported db type: %s for insert executor", execContext.DBType)
	return nil
}

func (e *ATExecutor) NewUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	switch execContext.DBType {
	case types.DBTypeMySQL:
		return mysql.NewUpdateExecutor(parserCtx, execContext, e.hooks)
	}
	log.Errorf("unsupported db type: %s for update executor", execContext.DBType)
	return nil
}

func (e *ATExecutor) NewDeleteExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	switch execContext.DBType {
	case types.DBTypeMySQL:
		return mysql.NewDeleteExecutor(parserCtx, execContext, e.hooks)
	}
	log.Errorf("unsupported db type: %s for delete executor", execContext.DBType)
	return nil
}

func (e *ATExecutor) NewSelectForUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	switch execContext.DBType {
	case types.DBTypeMySQL:
		return mysql.NewSelectForUpdateExecutor(parserCtx, execContext, e.hooks)
	}
	log.Errorf("unsupported db type: %s for select_for_update executor", execContext.DBType)
	return nil
}

func (e *ATExecutor) NewInsertOnUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	switch execContext.DBType {
	case types.DBTypeMySQL:
		return mysql.NewInsertOnUpdateExecutor(parserCtx, execContext, e.hooks)
	}
	log.Errorf("unsupported db type: %s for insert_on_update executor", execContext.DBType)
	return nil
}

func (e *ATExecutor) NewMultiExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	switch execContext.DBType {
	case types.DBTypeMySQL:
		return mysql.NewMultiExecutor(parserCtx, execContext, e.hooks)
	}
	log.Errorf("unsupported db type: %s for multi executor", execContext.DBType)
	return nil
}
