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

	"github.com/arana-db/parser/ast"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

var errMySQLATUnsupportedStatement = errors.New("seata AT: unsupported MySQL statement")

// validateMySQLATStatement prevents statements without Undo support from
// reaching the business callback unless their AST represents a read.
func validateMySQLATStatement(parsed *types.ParseContext) error {
	switch parsed.SQLType {
	case types.SQLTypeInsert, types.SQLTypeUpdate, types.SQLTypeDelete,
		types.SQLTypeInsertOnDuplicateUpdate, types.SQLTypeMulti:
		// The operation-specific executors validate these DML forms.
		return nil
	}

	stmt := parsed.UnclassifiedStmt
	guard := mysqlReadGuard{}
	if parsed.SelectStmt != nil {
		stmt = parsed.SelectStmt
		if parsed.SQLType == types.SQLTypeSelectForUpdate {
			guard.lockingRead = parsed.SelectStmt
		}
	}
	switch node := stmt.(type) {
	case *ast.SelectStmt, *ast.SetOprStmt, *ast.ShowStmt, *ast.ExplainForStmt:
	case *ast.ExplainStmt:
		if node.Analyze {
			return fmt.Errorf("%w: EXPLAIN ANALYZE executes its statement", errMySQLATUnsupportedStatement)
		}
		switch node.Stmt.(type) {
		case *ast.SelectStmt, *ast.SetOprStmt, *ast.ShowStmt, *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
			guard.explain = true
		default:
			return fmt.Errorf("%w: cannot explain %T in AT mode", errMySQLATUnsupportedStatement, node.Stmt)
		}
	default:
		return fmt.Errorf("%w: %T has no AT executor or read-only classification", errMySQLATUnsupportedStatement, stmt)
	}

	if _, ok := stmt.Accept(&guard); !ok {
		return fmt.Errorf("%w: %s", errMySQLATUnsupportedStatement, guard.reason)
	}
	return nil
}

type mysqlReadGuard struct {
	lockingRead *ast.SelectStmt
	explain     bool
	reason      string
}

func (g *mysqlReadGuard) Enter(node ast.Node) (ast.Node, bool) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		switch {
		case n.SelectIntoOpt != nil:
			g.reason = "SELECT INTO"
		case n.LockInfo != nil && n.LockInfo.LockType != ast.SelectLockNone && !g.explain:
			// The existing executor covers only top-level FOR UPDATE with a
			// table source and without an explicit list of lock targets.
			if n != g.lockingRead || n.Kind != ast.SelectStmtKindSelect || n.From == nil ||
				n.LockInfo.LockType != ast.SelectLockForUpdate || len(n.LockInfo.Tables) != 0 {
				g.reason = "unsupported SELECT locking clause"
			}
		}
	case *ast.VariableExpr:
		if n.Value != nil {
			g.reason = "variable assignment in a read"
		}
	case *ast.FuncCallExpr:
		if !isMySQLReadFunction(n) {
			g.reason = fmt.Sprintf("function %q is not recognized as read-only", n.FnName.O)
		}
	}
	return node, g.reason != ""
}

func (g *mysqlReadGuard) Leave(node ast.Node) (ast.Node, bool) {
	return node, g.reason == ""
}

func isMySQLReadFunction(fn *ast.FuncCallExpr) bool {
	// Qualified names can invoke stored functions even when their names
	// match built-ins. Generic unqualified calls also need an allowlist.
	if fn.Schema.O != "" {
		return false
	}
	switch fn.FnName.L {
	case ast.DateLiteral, ast.TimeLiteral, ast.TimestampLiteral:
		// Arana clears charset and collation on temporal literals. Ordinary
		// string arguments keep them even when a function uses the marker name.
		if len(fn.Args) != 1 {
			return false
		}
		value, ok := fn.Args[0].(ast.ValueExpr)
		if !ok {
			return false
		}
		_, isString := value.GetValue().(string)
		return isString && value.GetType().Charset == "" && value.GetType().Collate == ""
	case ast.Extract:
		if len(fn.Args) != 2 {
			return false
		}
		_, ok := fn.Args[0].(*ast.TimeUnitExpr)
		return ok
	case ast.CharFunc:
		// Arana appends a synthetic charset argument to CHAR. A call to
		// a routine named char_func has source positions for its arguments.
		if len(fn.Args) < 2 {
			return false
		}
		encoding, ok := fn.Args[len(fn.Args)-1].(ast.ValueExpr)
		if !ok || encoding.OriginTextPosition() != 0 {
			return false
		}
		_, isString := encoding.GetValue().(string)
		return encoding.GetValue() == nil ||
			(isString && encoding.GetType().Charset == "" && encoding.GetType().Collate == "")
	case ast.LastInsertId:
		return len(fn.Args) == 0
	case ast.Abs, ast.Ceil, ast.Ceiling, ast.Floor, ast.Round, ast.Truncate,
		ast.If, ast.Ifnull, ast.Nullif, "coalesce", ast.AnyValue, "greatest", "least",
		ast.Concat, ast.ConcatWS, ast.Length, ast.CharLength, ast.CharacterLength, ast.Hex,
		ast.Lower, ast.Lcase, ast.Upper, ast.Ucase, ast.Trim, ast.LTrim, ast.RTrim,
		ast.Left, ast.Right, ast.Substring, ast.Substr, ast.Replace,
		ast.Database, ast.Schema, ast.Version, ast.User, ast.CurrentUser,
		ast.ConnectionID, ast.FoundRows, ast.RowCount,
		ast.Now, ast.CurrentDate, ast.CurrentTime, ast.CurrentTimestamp, ast.Curdate, ast.Curtime,
		ast.Date, ast.DateAdd, ast.DateSub, ast.DateDiff, ast.DateFormat, ast.TimestampAdd, ast.TimestampDiff, ast.UnixTimestamp,
		ast.JSONExtract, ast.JSONUnquote, ast.JSONArray, ast.JSONObject, ast.JSONLength:
		return true
	default:
		return false
	}
}
