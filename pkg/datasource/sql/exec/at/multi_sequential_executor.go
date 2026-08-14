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
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

// execSequential executes validated statements in their original order.
//
// The first loop prepares all statement-local SQL and validates the total argument count.
// The second loop performs the actual business execution.
// It intentionally returns the final successful statement's ExecResult to
// match the aggregate path. RowsAffected and LastInsertId are not accumulated.
func (m *multiExecutor) execSequential(ctx context.Context, f exec.CallbackWithNamedValue, plan *multiExecutionPlan) (types.ExecResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("%w: execution plan is nil", errInvalidMultiSQL)
	}

	if len(plan.statements) == 0 {
		return nil, fmt.Errorf("%w: execution plan contains no statements", errInvalidMultiSQL)
	}

	queries := make([]string, len(plan.statements))
	argCounts := make([]int, len(plan.statements))
	statementParseContexts := make([]*types.ParseContext, len(plan.statements))
	totalArgCount := 0

	for index, statementCtx := range plan.statements {
		stmtNode, err := getStatementNode(statementCtx)
		if err != nil {
			return nil, fmt.Errorf("get statement %d AST: %w", index, err)
		}

		query, err := restoreStatementSQL(stmtNode)
		if err != nil {
			return nil, fmt.Errorf("restore statement %d SQL: %w", index, err)
		}

		// Multi-statement parsing assigns parameter-marker orders globally across
		// all child statements. Reparse each restored child as standalone SQL to
		// rebase marker orders to the child-local argument slice expected by the
		// existing single-statement executors.
		childParseCtx, err := parser.DoParser(query)
		if err != nil {
			return nil, fmt.Errorf("parse restored statement %d: %w", index, err)
		}

		if childParseCtx.ExecutorType != statementCtx.ExecutorType {
			return nil, fmt.Errorf(
				"%w: statement %d changed executor type from %v to %v after restoration",
				errInvalidMultiSQL, index, statementCtx.ExecutorType, childParseCtx.ExecutorType,
			)
		}

		childStmtNode, err := getStatementNode(childParseCtx)
		if err != nil {
			return nil, fmt.Errorf("get restored statement %d AST: %w", index, err)
		}

		argCount, err := countStatementParameters(childStmtNode)
		if err != nil {
			return nil, fmt.Errorf("count statement %d parameters: %w", index, err)
		}

		queries[index] = query
		argCounts[index] = argCount
		statementParseContexts[index] = childParseCtx
		totalArgCount += argCount
	}

	if totalArgCount != len(m.execContext.NamedValues) {
		return nil, fmt.Errorf(
			"%w: statements require %d arguments, but %d were provided",
			errInvalidMultiSQL, totalArgCount, len(m.execContext.NamedValues),
		)
	}

	if err := m.beforeHooks(ctx, m.execContext); err != nil {
		return nil, err
	}

	defer func() {
		m.afterHooks(ctx, m.execContext)
	}()

	var (
		argOffset  int
		lastResult types.ExecResult
	)

	for index, childParseCtx := range statementParseContexts {
		argCount := argCounts[index]
		statementArgs := cloneNamedValues(m.execContext.NamedValues[argOffset : argOffset+argCount])

		argOffset += argCount

		childExecCtx := *m.execContext
		childExecCtx.Query = queries[index]
		childExecCtx.ParseContext = childParseCtx
		childExecCtx.NamedValues = statementArgs
		childExecCtx.Values = nil

		childExecutor, err := newSequentialStatementExecutor(index, childParseCtx, &childExecCtx)
		if err != nil {
			return nil, err
		}

		result, err := childExecutor.ExecContext(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("execute statement %d: %w", index, err)
		}

		if result == nil {
			return nil, fmt.Errorf("%w: statement %d returned nil result", errInvalidMultiSQL, index)
		}

		lastResult = result
	}

	return lastResult, nil
}

// newSequentialStatementExecutor reuses the existing single-statement executors.
//
// Parent common and SQLTypeMulti hooks are owned by execSequential.
// Each child executor receives only hooks registered for its own SQL type.
func newSequentialStatementExecutor(index int, parseCtx *types.ParseContext, execCtx *types.ExecContext) (executor, error) {
	childHooks := hooksForSQLType(parseCtx.SQLType)
	switch parseCtx.ExecutorType {
	case types.InsertExecutor:
		return newInsertExecutor(parseCtx, execCtx, childHooks), nil

	case types.UpdateExecutor:
		return newUpdateExecutor(parseCtx, execCtx, childHooks), nil

	case types.DeleteExecutor:
		return newDeleteExecutor(parseCtx, execCtx, childHooks), nil

	default:
		return nil, fmt.Errorf("%w: statement %d uses executor type %v",
			errUnsupportedMultiSQL, index, parseCtx.ExecutorType,
		)
	}
}

// getStatementNode gets the concrete AST node from ParseContext.
func getStatementNode(parseCtx *types.ParseContext) (ast.StmtNode, error) {
	if parseCtx == nil {
		return nil, fmt.Errorf("%w: statement parse context is nil", errInvalidMultiSQL)
	}

	switch parseCtx.ExecutorType {
	case types.InsertExecutor:
		return parseCtx.InsertStmt, nil

	case types.UpdateExecutor:
		return parseCtx.UpdateStmt, nil

	case types.DeleteExecutor:
		return parseCtx.DeleteStmt, nil

	default:
		return nil, fmt.Errorf("%w: executor type %v", errUnsupportedMultiSQL, parseCtx.ExecutorType)
	}
}

// restoreStatementSQL obtains SQL that can be executed independently.
//
// Prefer the parser-preserved original statement text. If it is unavailable,
// restore SQL from the AST.
func restoreStatementSQL(stmt ast.StmtNode) (string, error) {
	if stmt == nil {
		return "", fmt.Errorf("%w: statement AST is nil", errInvalidMultiSQL)
	}

	query := trimStatementSemicolon(stmt.OriginalText())
	if query != "" {
		return query, nil
	}

	var buffer bytes.Buffer
	restoreCtx := format.NewRestoreCtx(format.DefaultRestoreFlags, &buffer)

	if err := stmt.Restore(restoreCtx); err != nil {
		return "", err
	}

	query = trimStatementSemicolon(buffer.String())
	if query == "" {
		return "", fmt.Errorf("%w: restored statement SQL is empty", errInvalidMultiSQL)
	}
	return query, nil
}

func trimStatementSemicolon(query string) string {
	query = strings.TrimSpace(query)

	for strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	return query
}

type parameterCounter struct {
	count int
}

func (c *parameterCounter) Enter(node ast.Node) (ast.Node, bool) {
	if _, ok := node.(ast.ParamMarkerExpr); ok {
		c.count++
	}
	return node, false
}

func (c *parameterCounter) Leave(node ast.Node) (ast.Node, bool) {
	return node, true
}

func countStatementParameters(stmt ast.StmtNode) (int, error) {
	counter := new(parameterCounter)

	if _, ok := stmt.Accept(counter); !ok {
		return 0, fmt.Errorf("%w: parameter traversal stopped unexpectedly", errInvalidMultiSQL)
	}

	return counter.count, nil
}

func cloneNamedValues(values []driver.NamedValue) []driver.NamedValue {
	cloned := make([]driver.NamedValue, len(values))
	copy(cloned, values)
	for index := range cloned {
		cloned[index].Ordinal = index + 1
	}
	return cloned
}
