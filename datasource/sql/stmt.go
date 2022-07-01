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
)

type Stmt struct {
	ctx    *TransactionContext
	hooks  []SQLHook
	query  string
	target *gosql.Stmt
}

// ExecContext
func (s *Stmt) ExecContext(ctx context.Context, args ...interface{}) (gosql.Result, error) {
	executor, err := buildExecutor(s.query)
	if err != nil {
		return nil, err
	}

	ret, err := executor.Exec(func(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
		return s.target.ExecContext(ctx, args...)
	})

	if err != nil {
		return nil, err
	}

	return ret.(gosql.Result), nil
}

// QueryContext
func (s *Stmt) QueryContext(ctx context.Context, args ...interface{}) (*gosql.Rows, error) {
	return s.target.QueryContext(ctx, args)
}

// QueryRowContext
func (s *Stmt) QueryRowContext(ctx context.Context, args ...interface{}) *gosql.Row {
	return s.target.QueryRowContext(ctx, args)
}
