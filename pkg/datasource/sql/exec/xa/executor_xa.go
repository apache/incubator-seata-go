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

package xa

import (
	"context"
	"database/sql/driver"

	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

// XAExecutor The XA transaction manager.
type XAExecutor struct {
	hooks []exec.SQLHook
	ex    exec.SQLExecutor
}

// Interceptors set xa executor hooks
func (e *XAExecutor) Interceptors(hooks []exec.SQLHook) {
	e.hooks = hooks
}

// ExecWithNamedValue
func (e *XAExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	for _, hook := range e.hooks {
		hook.Before(ctx, execCtx)
	}

	defer func() {
		for _, hook := range e.hooks {
			hook.After(ctx, execCtx)
		}
	}()

	if e.ex != nil {
		return e.ex.ExecWithNamedValue(ctx, execCtx, f)
	}

	return f(ctx, execCtx.Query, execCtx.NamedValues)
}

// ExecWithValue
func (e *XAExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	for _, hook := range e.hooks {
		hook.Before(ctx, execCtx)
	}

	defer func() {
		for _, hook := range e.hooks {
			hook.After(ctx, execCtx)
		}
	}()

	if e.ex != nil {
		return e.ex.ExecWithValue(ctx, execCtx, f)
	}

	nvargs := make([]driver.NamedValue, len(execCtx.Values))
	for i, value := range execCtx.Values {
		nvargs = append(nvargs, driver.NamedValue{
			Value:   value,
			Ordinal: i,
		})
	}
	execCtx.NamedValues = nvargs

	return f(ctx, execCtx.Query, execCtx.NamedValues)
}
