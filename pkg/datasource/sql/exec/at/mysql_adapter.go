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
	"reflect"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	"github.com/pkg/errors"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

var parseSQLQuery = func(query string) (*types.ParseContext, error) {
	return parser.ParseSQLForDB(query, types.DBTypeMySQL)
}

// mysqlAdapter selects a statement executor. Hooks, ASTs and image state
// belong to the current execution.
type mysqlAdapter struct{}

func (a mysqlAdapter) execute(ctx context.Context, execCtx *types.ExecContext, parsed *types.ParseContext, hooks []exec.SQLHook, run exec.CallbackWithNamedValue) (types.ExecResult, error) {
	isATMode := execCtx.TxCtx.IsATMode()
	if !isATMode && !execCtx.IsRequireGlobalLock {
		return newPlainExecutor(nil, execCtx).ExecContext(ctx, run)
	}
	if parsed == nil {
		var err error
		parsed, err = parseSQLQuery(execCtx.Query)
		if err != nil {
			return nil, err
		}
	}
	if !isATMode && parsed.SQLType != types.SQLTypeSelectForUpdate {
		return newPlainExecutor(parsed, execCtx).ExecContext(ctx, run)
	}
	if parsed.ExecutorType == types.ReplaceIntoExecutor {
		return nil, errors.New("NotSupportYetException: AT mode currently does not support REPLACE INTO statement")
	}
	if isATMode {
		if err := validateMySQLATStatement(parsed); err != nil {
			return nil, err
		}
	}

	switch parsed.SQLType {
	case types.SQLTypeInsert, types.SQLTypeUpdate, types.SQLTypeDelete,
		types.SQLTypeSelectForUpdate, types.SQLTypeInsertOnDuplicateUpdate, types.SQLTypeMulti:
		if err := parser.BindTableRefs(parsed); err != nil {
			return nil, err
		}
	}
	execCtx.DBType = types.DBTypeMySQL
	return a.newStatementExecutor(parsed, execCtx, hooks).ExecContext(ctx, run)
}

// newStatementExecutor is used by the root and by validated children of a
// sequential batch. Each caller supplies the hooks for its own statement.
func (mysqlAdapter) newStatementExecutor(parsed *types.ParseContext, execCtx *types.ExecContext, hooks []exec.SQLHook) executor {
	switch parsed.SQLType {
	case types.SQLTypeInsert:
		return newInsertExecutor(parsed, execCtx, hooks)
	case types.SQLTypeUpdate:
		return newUpdateExecutor(parsed, execCtx, hooks)
	case types.SQLTypeDelete:
		return newDeleteExecutor(parsed, execCtx, hooks)
	case types.SQLTypeSelectForUpdate:
		return newSelectForUpdateExecutor(parsed, execCtx, hooks)
	case types.SQLTypeInsertOnDuplicateUpdate:
		return newInsertOnUpdateExecutor(parsed, execCtx, hooks)
	case types.SQLTypeMulti:
		return newMultiExecutor(parsed, execCtx, hooks)
	default:
		return newPlainExecutor(parsed, execCtx)
	}
}

func (mysqlAdapter) qualifiedTableName(key *types.TableMetaKey, fallback string) string {
	return quoteQualifiedTable(key.DBName, key.TableName, fallback, "`")
}

func (mysqlAdapter) quoteImagePK(column string) string {
	return "`" + column + "`"
}

func (mysqlAdapter) restoreImageSelectSQL(stmt *ast.SelectStmt) (string, error) {
	var sql strings.Builder
	if err := stmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &sql)); err != nil {
		return "", err
	}
	// MySQL uses the positional markers restored by the AST.
	return sql.String(), nil
}

func (mysqlAdapter) buildDMLLockKey(image *types.RecordImage, meta types.TableMeta) string {
	return util.BuildLockKey(image, meta)
}

// buildSelectLockKey keeps the table spelling and scan representation used by
// locking reads. PostgreSQL currently shares this encoding.
func (mysqlAdapter) buildSelectLockKey(rows driver.Rows, meta *types.TableMeta) (string, error) {
	var lockKeys bytes.Buffer
	columnNames := meta.GetPrimaryKeyOnlyName()
	sqlRows := util.NewScanRows(rows)
	for sqlRows.Next() {
		values := getScanSlice(columnNames, meta)
		if err := sqlRows.Scan(values...); err != nil {
			return "", err
		}
		if lockKeys.Len() == 0 {
			lockKeys.WriteString(meta.TableName)
			lockKeys.WriteByte(':')
		} else {
			lockKeys.WriteByte(',')
		}
		for i, value := range values {
			if i > 0 {
				lockKeys.WriteByte('_')
			}
			// Nullable scan destinations expose their underlying value via Value.
			ty := reflect.TypeOf(value)
			if method, ok := ty.MethodByName("Value"); ok {
				result := method.Func.Call([]reflect.Value{reflect.ValueOf(value)})
				if result[1].IsNil() {
					lockKeys.WriteString(result[0].Elem().String())
				}
				continue
			}
			fmt.Fprint(&lockKeys, reflect.ValueOf(value).Elem())
		}
	}
	if err := sqlRows.Err(); err != nil {
		return "", err
	}
	return lockKeys.String(), nil
}
