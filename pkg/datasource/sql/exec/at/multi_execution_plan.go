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
	"errors"
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

var (
	errInvalidMultiSQL     = errors.New("invalid multi SQL")
	errUnsupportedMultiSQL = errors.New("unsupported multi SQL")
)

// multiExecutionPlan contains the validated statements in their original order
// and records whether they can use the existing aggregate execution path.
type multiExecutionPlan struct {
	statements       []*types.ParseContext
	useAggregatePath bool
}

// buildMultiExecutionPlan validates all statements before any before-image query
// or business SQL execution occurs.
func buildMultiExecutionPlan(parseCtx *types.ParseContext, dbType types.DBType) (*multiExecutionPlan, error) {
	if parseCtx == nil {
		return nil, fmt.Errorf("%w: parse context", errInvalidMultiSQL)
	}

	if len(parseCtx.MultiStmt) < 2 {
		return nil, fmt.Errorf(
			"%w: expected at least two statements, got %d",
			errInvalidMultiSQL, len(parseCtx.MultiStmt))
	}

	// Multi-SQL AT execution is currently enabled only for MySQL.
	if effectiveDBType(dbType) != types.DBTypeMySQL {
		return nil, fmt.Errorf(
			"%w: database type %v is not supported",
			errUnsupportedMultiSQL, dbType,
		)
	}

	statements := append([]*types.ParseContext(nil), parseCtx.MultiStmt...)
	tableNames := make([]string, len(statements))

	for index, statementCtx := range statements {
		tableName, err := validateMultiStatement(index, statementCtx)
		if err != nil {
			return nil, err
		}
		tableNames[index] = tableName
	}

	return &multiExecutionPlan{
		statements:       statements,
		useAggregatePath: canUseAggregateFastPath(statements, tableNames),
	}, nil
}

// validateMultiStatement validates one parsed DML statement.
//
// The table name is returned temporarily for aggregate-path detection.
func validateMultiStatement(index int, parseCtx *types.ParseContext) (string, error) {
	if parseCtx == nil {
		return "", fmt.Errorf("%w: statement %d parse context is nil", errInvalidMultiSQL, index)
	}

	switch parseCtx.ExecutorType {
	case types.InsertExecutor:
		if parseCtx.InsertStmt == nil {
			return "", fmt.Errorf("%w: statement %d is marked as INSERT but has no INSERT AST", errInvalidMultiSQL, index)
		}

	case types.UpdateExecutor:
		if parseCtx.UpdateStmt == nil {
			return "", fmt.Errorf("%w: statement %d is marked as UPDATE but has no UPDATE AST", errInvalidMultiSQL, index)
		}

		updateStmt := parseCtx.UpdateStmt
		if updateStmt.TableRefs == nil || updateStmt.TableRefs.TableRefs == nil {
			return "", fmt.Errorf("%w: statement %d has invalid UPDATE table references", errInvalidMultiSQL, index)
		}

		if updateStmt.TableRefs.TableRefs.Right != nil {
			return "", fmt.Errorf("%w: statement %d uses UPDATE JOIN", errUnsupportedMultiSQL, index)
		}

	case types.DeleteExecutor:
		if parseCtx.DeleteStmt == nil {
			return "", fmt.Errorf("%w: statement %d is marked as DELETE but has no DELETE AST", errInvalidMultiSQL, index)
		}

		if parseCtx.DeleteStmt.IsMultiTable {
			return "", fmt.Errorf("%w: statement %d uses multi-table DELETE", errUnsupportedMultiSQL, index)
		}

	default:
		return "", fmt.Errorf("%w: statement %d uses executor type %v", errUnsupportedMultiSQL, index, parseCtx.ExecutorType)
	}

	tableName, err := parseCtx.GetTableName()
	if err != nil {
		return "", fmt.Errorf("%w: get table name for statement %d: %w", errInvalidMultiSQL, index, err)
	}

	return tableName, nil
}

// canUseAggregateFastPath reports whether all statements satisfy the static
// contract of the existing aggregate UPDATE/DELETE executors.
//
// Aggregate execution is an optimization. Any statement that cannot be proven
// safe must use the sequential single-statement executors.
func canUseAggregateFastPath(statements []*types.ParseContext, tableNames []string) bool {
	if len(statements) < 2 || len(statements) != len(tableNames) {
		return false
	}

	firstStatement := statements[0]
	if firstStatement == nil {
		return false
	}

	if firstStatement.ExecutorType != types.UpdateExecutor && firstStatement.ExecutorType != types.DeleteExecutor {
		return false
	}

	firstTableName := tableNames[0]

	for index, statementCtx := range statements {
		if statementCtx == nil {
			return false
		}

		if statementCtx.ExecutorType != firstStatement.ExecutorType {
			return false
		}

		if tableNames[index] != firstTableName {
			return false
		}

		statementNode, err := getStatementNode(statementCtx)
		if err != nil {
			return false
		}

		parameterCount, err := countStatementParameters(statementNode)
		if err != nil || parameterCount != 0 {
			return false
		}

		switch statementCtx.ExecutorType {
		case types.UpdateExecutor:
			updateStmt := statementCtx.UpdateStmt
			if updateStmt == nil || updateStmt.Where == nil || updateStmt.Limit != nil || updateStmt.Order != nil {
				return false
			}

			// UPDATE JOIN uses the single-statement update-join executor.
			if updateStmt.TableRefs == nil || updateStmt.TableRefs.TableRefs == nil || updateStmt.TableRefs.TableRefs.Right != nil {
				return false
			}

		case types.DeleteExecutor:
			deleteStmt := statementCtx.DeleteStmt
			if deleteStmt == nil || deleteStmt.IsMultiTable || deleteStmt.Where == nil || deleteStmt.Limit != nil || deleteStmt.Order != nil {
				return false
			}

		default:
			return false
		}
	}

	return true
}
