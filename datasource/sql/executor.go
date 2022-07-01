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

	"github.com/seata/seata-go-datasource/sql/parser"
)

var (
	executorSolts = make(map[DBType]map[parser.ExecutorType]SQLExecutor)
)

type (
	callback func(ctx context.Context, query string, args ...interface{}) (interface{}, error)

	ExecResult struct {
		Result gosql.Result
		Row    *gosql.Row
		Rows   *gosql.Rows
	}
)

// SQLExecutor
type SQLExecutor interface {

	// Exec
	Exec(f callback) (interface{}, error)
}

// buildExecutor
func buildExecutor(query string) (SQLExecutor, error) {
	return nil, nil
}
