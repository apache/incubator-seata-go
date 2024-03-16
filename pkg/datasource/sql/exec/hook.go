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

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	commonHook = make([]SQLHook, 0, 4)
	// todo support distinguish between different db type
	hookSolts = map[types.SQLType][]SQLHook{}
)

// RegisterCommonHook not goroutine safe
func RegisterCommonHook(hook SQLHook) {
	commonHook = append(commonHook, hook)
}

func CleanCommonHook() {
	commonHook = make([]SQLHook, 0, 4)
}

// RegisterHook not goroutine safe
func RegisterHook(hook SQLHook) {
	sqlType := hook.Type()
	if sqlType == types.SQLTypeUnknown {
		return
	}

	_, ok := hookSolts[sqlType]
	if !ok {
		hookSolts[sqlType] = make([]SQLHook, 0, 4)
	}
	hookSolts[sqlType] = append(hookSolts[sqlType], hook)
}

// SQLHook SQL execution front and back interceptor
// case 1. Used to intercept SQL to achieve the generation of front and rear mirrors
// case 2. Burning point to report
// case 3. SQL black and white list
type SQLHook interface {
	Type() types.SQLType
	Before(ctx context.Context, execCtx *types.ExecContext) error
	After(ctx context.Context, execCtx *types.ExecContext) error
}
