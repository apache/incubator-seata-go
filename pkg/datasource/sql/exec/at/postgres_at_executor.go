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
	"errors"
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

// ErrPostgreSQLATUnsupported indicates the PostgreSQL AT executor root is wired
// but the requested SQL type is still outside the current PostgreSQL AT slice.
var ErrPostgreSQLATUnsupported = errors.New("postgresql AT global execution is not supported yet")

type postgresATExecutor struct {
	hooks []exec.SQLHook
}

func (e *postgresATExecutor) Interceptors(hooks []exec.SQLHook) {
	e.hooks = hooks
}

func (e *postgresATExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if !isGlobalATExecution(ctx, execCtx) {
		return newPlainExecutor(nil, execCtx).ExecContext(ctx, f)
	}

	queryParser, err := parseSQLQuery(execCtx.Query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgreSQLATUnsupported, err)
	}

	var executor executor
	switch queryParser.SQLType {
	case types.SQLTypeInsert:
		executor = newInsertExecutor(queryParser, execCtx, e.hooks)
	case types.SQLTypeUpdate:
		executor = newUpdateExecutor(queryParser, execCtx, e.hooks)
	case types.SQLTypeDelete:
		executor = newDeleteExecutor(queryParser, execCtx, e.hooks)
	case types.SQLTypeSelectForUpdate:
		executor = newSelectForUpdateExecutor(queryParser, execCtx, e.hooks)
	default:
		return nil, fmt.Errorf("%w: sqlType=%v", ErrPostgreSQLATUnsupported, queryParser.SQLType)
	}

	return executor.ExecContext(ctx, f)
}

func (e *postgresATExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	execCtx.NamedValues = util.ValueToNamedValue(execCtx.Values)
	return e.ExecWithNamedValue(ctx, execCtx, f)
}

func isGlobalATExecution(ctx context.Context, execCtx *types.ExecContext) bool {
	if execCtx != nil && execCtx.TxCtx != nil && execCtx.TxCtx.OpenGlobalTransaction() {
		return true
	}

	return isGlobalTx(ctx)
}
