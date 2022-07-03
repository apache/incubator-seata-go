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

package sql

import (
	"context"
	gosql "database/sql"

	"github.com/seata/seata-go-datasource/sql/exec"
	"github.com/seata/seata-go-datasource/sql/types"
)

type Stmt struct {
	ctx    *types.TransactionContext
	query  string
	target *gosql.Stmt
}

// ExecContext
func (s *Stmt) ExecContext(ctx context.Context, args ...interface{}) (gosql.Result, error) {
	if s.ctx == nil {
		return s.target.ExecContext(ctx, args...)
	}

	executor, err := exec.BuildExecutor(s.ctx.DBType, s.query)
	if err != nil {
		return nil, err
	}

	ret, err := executor.Exec(s.ctx, func(ctx context.Context, query string, args ...interface{}) (types.ExecResult, error) {
		ret, err := s.target.ExecContext(ctx, args...)
		if err != nil {
			return nil, err
		}
		return types.NewResult(types.WithResult(ret)), nil
	})

	if err != nil {
		return nil, err
	}

	return ret.GetResult(), nil
}

// QueryContext
func (s *Stmt) QueryContext(ctx context.Context, args ...interface{}) (*gosql.Rows, error) {
	return s.target.QueryContext(ctx, args)
}

// QueryRowContext
func (s *Stmt) QueryRowContext(ctx context.Context, args ...interface{}) *gosql.Row {
	return s.target.QueryRowContext(ctx, args)
}
