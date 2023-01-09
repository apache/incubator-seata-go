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

	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/tm"

	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

func Init() {
	exec.RegisterATExecutor(types.DBTypeMySQL, func() exec.SQLExecutor { return &AtExecutor{} })
}

type executor interface {
	ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error)
}

type AtExecutor struct {
	hooks []exec.SQLHook
}

func (e *AtExecutor) Interceptors(hooks []exec.SQLHook) {
	e.hooks = hooks
}

// ExecWithNamedValue
func (e *AtExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	parser, err := parser.DoParser(execCtx.Query)
	if err != nil {
		return nil, err
	}

	var exec executor

	if !tm.IsGlobalTx(ctx) {
		exec = NewPlainExecutor(parser, execCtx)
	} else {
		switch parser.SQLType {
		case types.SQLTypeInsert:
			exec = NewInsertExecutor(parser, execCtx, e.hooks)
		case types.SQLTypeUpdate:
			exec = NewUpdateExecutor(parser, execCtx, e.hooks)
		case types.SQLTypeDelete:
			exec = NewDeleteExecutor(parser, execCtx, e.hooks)
		//case types.SQLTypeSelectForUpdate:
		//case types.SQLTypeMultiDelete:
		//case types.SQLTypeMultiUpdate:
		default:
			exec = NewPlainExecutor(parser, execCtx)
		}
	}

	return exec.ExecContext(ctx, f)
}

// ExecWithValue
func (e *AtExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	execCtx.NamedValues = util.ValueToNamedValue(execCtx.Values)
	return e.ExecWithNamedValue(ctx, execCtx, f)
}
