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
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"time"

	"seata.apache.org/seata-go/pkg/tm"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/builder"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	seatabytes "seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	// todo replace by config
	retryTimes    = 5
	retryInterval = 20 * time.Millisecond
)

type SelectForUpdateExecutor struct {
	builder.BasicUndoLogBuilder
	dbType types.DBType
}

func (s SelectForUpdateExecutor) interceptors(interceptors []SQLHook) {
}

func (s *SelectForUpdateExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
	s.dbType = execCtx.DBType
	if !tm.IsGlobalTx(ctx) && !execCtx.IsRequireGlobalLock {
		return f(ctx, execCtx.Query, execCtx.NamedValues)
	}

	var (
		tx                 driver.Tx
		nowTs              = time.Now().Unix()
		result             types.ExecResult
		savepointName      string
		originalAutoCommit = execCtx.IsAutoCommit
	)

	table, err := execCtx.ParseContext.GetTableName()
	if err != nil {
		return nil, err
	}
	// build query primary key sql
	selectPKSQL, err := s.buildSelectPKSQL(execCtx.ParseContext.SelectStmt, execCtx.MetaDataMap[table])
	if err != nil {
		return nil, err
	}

	i := 0
	for ; i < retryTimes; i++ {
		if originalAutoCommit {
			// In order to hold the local db lock during global lock checking
			// set auto commit value to false first if original auto commit was true
			tx, err = execCtx.Conn.Begin()
			if err != nil {
				return nil, err
			}
			execCtx.IsAutoCommit = false
		} else if execCtx.IsSupportsSavepoints {
			// In order to release the local db lock when global lock conflict
			// create a save point if original auto commit was false, then use the save point here to release db
			// lock during global lock checking if necessary
			savepointName = fmt.Sprintf("savepoint %d;", nowTs)
			stmt, err := execCtx.Conn.Prepare(savepointName)
			if err != nil {
				return nil, err
			}
			if _, err = stmt.Exec(nil); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("not support savepoint. please check your db version")
		}

		// execute business SQL, try to get local lock
		result, err = f(ctx, execCtx.Query, execCtx.NamedValues)
		if err != nil {
			return nil, err
		}

		// query primary key values
		stmt, err := execCtx.Conn.Prepare(selectPKSQL)
		if err != nil {
			return nil, err
		}
		values := make([]driver.Value, 0, len(execCtx.NamedValues))
		for _, val := range execCtx.NamedValues {
			values = append(values, val.Value)
		}
		rows, err := stmt.Query(values)
		if err != nil {
			return nil, err
		}

		lockKey := s.buildLockKey(rows, execCtx.MetaDataMap[table])
		if lockKey == "" {
			break
		}
		// check global lock
		lockable, err := datasource.GetDataSourceManager(branch.BranchTypeAT).LockQuery(ctx, rm.LockQueryParam{
			Xid:        execCtx.TxCtx.XID,
			BranchType: branch.BranchTypeAT,
			ResourceId: execCtx.TxCtx.ResourceID,
			LockKeys:   lockKey,
		})

		// if obtained global lock
		if err == nil && lockable {
			break
		}

		if savepointName != "" {
			if stmt, err = execCtx.Conn.Prepare(fmt.Sprintf("rollback to %s;", savepointName)); err != nil {
				return nil, err
			}
			if _, err = stmt.Exec(nil); err != nil {
				return nil, err
			}
		} else {
			if err = tx.Rollback(); err != nil {
				return nil, err
			}
		}
		time.Sleep(retryInterval)
	}

	if i >= retryTimes {
		return nil, fmt.Errorf("global lock wait timeout")
	}

	if originalAutoCommit {
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		execCtx.IsAutoCommit = true
	}
	return result, nil
}

func (s *SelectForUpdateExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithValue) (types.ExecResult, error) {
	s.dbType = execCtx.DBType
	if !tm.IsGlobalTx(ctx) && !execCtx.IsRequireGlobalLock {
		return f(ctx, execCtx.Query, execCtx.Values)
	}

	var (
		tx                 driver.Tx
		nowTs              = time.Now().Unix()
		result             types.ExecResult
		savepointName      string
		originalAutoCommit = execCtx.IsAutoCommit
	)

	table, err := execCtx.ParseContext.GetTableName()
	if err != nil {
		return nil, err
	}
	// build query primary key sql
	selectPKSQL, err := s.buildSelectPKSQL(execCtx.ParseContext.SelectStmt, execCtx.MetaDataMap[table])
	if err != nil {
		return nil, err
	}

	i := 0
	for ; i < retryTimes; i++ {
		if originalAutoCommit {
			// In order to hold the local db lock during global lock checking
			// set auto commit value to false first if original auto commit was true
			tx, err = execCtx.Conn.Begin()
			if err != nil {
				return nil, err
			}
		} else if execCtx.IsSupportsSavepoints {
			// In order to release the local db lock when global lock conflict
			// create a save point if original auto commit was false, then use the save point here to release db
			// lock during global lock checking if necessary
			savepointName = fmt.Sprintf("savepoint %d;", nowTs)
			stmt, err := execCtx.Conn.Prepare(savepointName)
			if err != nil {
				return nil, err
			}
			if _, err = stmt.Exec(nil); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("not support savepoint. please check your db version")
		}

		// execute business SQL, try to get local lock
		result, err = f(ctx, execCtx.Query, execCtx.Values)
		if err != nil {
			return nil, err
		}

		// query primary key values
		stmt, err := execCtx.Conn.Prepare(selectPKSQL)
		if err != nil {
			return nil, err
		}
		rows, err := stmt.Query(execCtx.Values)
		if err != nil {
			return nil, err
		}

		lockKey := s.buildLockKey(rows, execCtx.MetaDataMap[table])
		if lockKey == "" {
			break
		}
		// check global lock
		lockable, err := datasource.GetDataSourceManager(branch.BranchTypeAT).LockQuery(ctx, rm.LockQueryParam{
			Xid:        execCtx.TxCtx.XID,
			BranchType: branch.BranchTypeAT,
			ResourceId: execCtx.TxCtx.ResourceID,
			LockKeys:   lockKey,
		})

		// has obtained global lock
		if err == nil && lockable {
			break
		}

		if savepointName != "" {
			if stmt, err = execCtx.Conn.Prepare(fmt.Sprintf("rollback to %s;", savepointName)); err != nil {
				return nil, err
			}
			if _, err = stmt.Exec(nil); err != nil {
				return nil, err
			}
		} else {
			if err = tx.Rollback(); err != nil {
				return nil, err
			}
		}
		time.Sleep(retryInterval)
	}

	if i >= retryTimes {
		return nil, fmt.Errorf("global lock wait timeout")
	}

	if originalAutoCommit {
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		execCtx.IsAutoCommit = true
	}
	return result, nil
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *SelectForUpdateExecutor) buildSelectPKSQL(stmt *ast.SelectStmt, meta types.TableMeta) (string, error) {
	pks := meta.GetPrimaryKeyOnlyName()
	if len(pks) == 0 {
		return "", fmt.Errorf("%s needs to contain the primary key.", meta.TableName)
	}

	fields := []*ast.SelectField{}
	for _, column := range pks {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.CIStr{
						O: column,
						L: strings.ToLower(column),
					},
				},
			},
		})
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           stmt.From,
		Where:          stmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		OrderBy:        stmt.OrderBy,
		Limit:          stmt.Limit,
		TableHints:     stmt.TableHints,
		LockInfo:       stmt.LockInfo,
	}

	b := seatabytes.NewByteBuffer([]byte{})
	selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	sql = u.adaptSQLSyntax(sql)
	log.Infof("build select sql by update sourceQuery, sql {}", sql)

	return sql, nil
}

// the string as local key. the local key example(multi pk): "t_user:1_a,2_b"
func (s *SelectForUpdateExecutor) buildLockKey(rows driver.Rows, meta types.TableMeta) string {
	var (
		lockKeys      bytes.Buffer
		filedSequence int
	)
	lockKeys.WriteString(meta.TableName)
	lockKeys.WriteString(":")

	ss := s.GetScanSlice(meta.GetPrimaryKeyOnlyName(), &meta)
	for {
		err := rows.Next(ss)
		if err == io.EOF {
			break
		}

		if filedSequence > 0 {
			lockKeys.WriteString(",")
		}

		pkSplitIndex := 0
		for _, value := range ss {
			if pkSplitIndex > 0 {
				lockKeys.WriteString("_")
			}
			lockKeys.WriteString(fmt.Sprintf("%v", value))
			pkSplitIndex++
		}
		filedSequence++
	}
	return lockKeys.String()
}

func (s *SelectForUpdateExecutor) escapeIdentifier(name string) string {
	if name == "" {
		return name
	}
	switch s.dbType {
	case types.DBTypeMySQL:
		if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
			return name
		}
		return "`" + name + "`"
	case types.DBTypePostgreSQL:
		if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
			return name
		}
		return "\"" + name + "\""
	default:
		return name
	}
}

func (s *SelectForUpdateExecutor) adaptSQLSyntax(sql string) string {
	switch s.dbType {
	case types.DBTypePostgreSQL:
		return s.adaptPostgreSQLSyntax(sql)
	case types.DBTypeMySQL:
		return s.adaptMySQLSyntax(sql)
	}
	return sql
}

func (s *SelectForUpdateExecutor) adaptPostgreSQLSyntax(sql string) string {
	sql = strings.ReplaceAll(sql, "SQL_NO_CACHE ", "")
	sql = strings.ReplaceAll(sql, "SQL_CACHE ", "")
	sql = strings.ReplaceAll(sql, "`", "\"")

	paramCounter := 1
	for strings.Contains(sql, "?") {
		sql = strings.Replace(sql, "?", fmt.Sprintf("$%d", paramCounter), 1)
		paramCounter++
	}

	return sql
}

func (s *SelectForUpdateExecutor) adaptMySQLSyntax(sql string) string {
	return sql
}

func (u *SelectForUpdateExecutor) buildSelectPKSQLForPostgreSQL(stmt *tree.Select, meta types.TableMeta) (string, error) {
	pks := meta.GetPrimaryKeyOnlyName()
	if len(pks) == 0 {
		return "", fmt.Errorf("%s needs to contain the primary key.", meta.TableName)
	}

	selectList := make(tree.SelectExprs, 0, len(pks))
	for _, column := range pks {
		escapedColumn := u.escapeIdentifier(column)
		selectList = append(selectList, tree.SelectExpr{
			Expr: &tree.UnresolvedName{
				NumParts: 1,
				Parts:    tree.NameParts{escapedColumn},
			},
		})
	}

	selectClause := &tree.SelectClause{
		Exprs: selectList,
		From:  stmt.Select.(*tree.SelectClause).From,
		Where: stmt.Select.(*tree.SelectClause).Where,
	}

	sql := tree.AsString(&tree.Select{
		Select:  selectClause,
		With:    stmt.With,
		OrderBy: stmt.OrderBy,
		Limit:   stmt.Limit,
		Locking: stmt.Locking,
	})

	sql = u.adaptSQLSyntax(sql)
	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	return sql, nil
}
