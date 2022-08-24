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

	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

var (
	commonHook = make([]SQLInterceptor, 0, 4)
	// todo support distinguish between different db type
	hookSolts = map[types.SQLType][]SQLInterceptor{}
)

// RegisCommonHook not goroutine safe
func RegisCommonHook(hook SQLInterceptor) {
	commonHook = append(commonHook, hook)
}

// RegisHook not goroutine safe
func RegisHook(hook SQLInterceptor) {
	_, ok := hookSolts[hook.Type()]

	if !ok {
		hookSolts[hook.Type()] = make([]SQLInterceptor, 0, 4)
	}

	hookSolts[hook.Type()] = append(hookSolts[hook.Type()], hook)
}

// ExecContext
type ExecContext struct {
	TxCtx       *types.TransactionContext
	Query       string
	NamedValues []driver.NamedValue
	Values      []driver.Value
}

// SQLHook SQL execution front and back interceptor
// case 1. Used to intercept SQL to achieve the generation of front and rear mirrors
// case 2. Burning point to report
// case 3. SQL black and white list
type SQLInterceptor interface {
	Type() types.SQLType

	// Before
	Before(ctx context.Context, execCtx *ExecContext)

	// After
	After(ctx context.Context, execCtx *ExecContext)
}
