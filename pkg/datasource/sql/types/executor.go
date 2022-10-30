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

package types

import "github.com/arana-db/parser/ast"

// ExecutorType
//
//go:generate stringer -type=ExecutorType
type ExecutorType int32

const (
	_ ExecutorType = iota
	UnSupportExecutor
	InsertExecutor
	UpdateExecutor
	DeleteExecutor
	ReplaceIntoExecutor
	MultiExecutor
	InsertOnDuplicateExecutor
)

type ParseContext struct {
	// SQLType
	SQLType SQLType
	// ExecutorType
	ExecutorType ExecutorType
	// InsertStmt
	InsertStmt *ast.InsertStmt
	// UpdateStmt
	UpdateStmt *ast.UpdateStmt
	// DeleteStmt
	DeleteStmt *ast.DeleteStmt
	MultiStmt  []*ParseContext
}

func (p *ParseContext) HasValidStmt() bool {
	return p.InsertStmt != nil || p.UpdateStmt != nil || p.DeleteStmt != nil
}
