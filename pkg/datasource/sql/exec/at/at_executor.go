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

	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/tm"
)

func Init() {
	exec.RegisterATExecutor(types.DBTypeMySQL, func() exec.SQLExecutor { return &ATExecutor{} })
}

type executor interface {
	ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error)
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

	var executor executor

	if !tm.IsGlobalTx(ctx) {
		executor = NewPlainExecutor(queryParser, execCtx)
	} else {
		switch queryParser.SQLType {
		case types.SQLTypeInsert:
			executor = NewInsertExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeUpdate:
			executor = NewUpdateExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeDelete:
			executor = NewDeleteExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeSelectForUpdate:
			executor = NewSelectForUpdateExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeInsertOnUpdate:
			executor = NewInsertOnUpdateExecutor(queryParser, execCtx, e.hooks)
		case types.SQLTypeMulti:
			executor = NewMultiExecutor(queryParser, execCtx, e.hooks)
		default:
			executor = NewPlainExecutor(queryParser, execCtx)
		}
	}

	return executor.ExecContext(ctx, f)
}

// ExecWithValue transfer value to nameValue execute
func (e *ATExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	execCtx.NamedValues = util.ValueToNamedValue(execCtx.Values)
	return e.ExecWithNamedValue(ctx, execCtx, f)
}
