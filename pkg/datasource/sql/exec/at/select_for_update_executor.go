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
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"time"

	"seata.apache.org/seata-go/pkg/tm"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/util/backoff"
	seatabytes "seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	lockConflictError = errors.New("lock conflict error")
)

type selectForUpdateExecutor struct {
	baseExecutor

	parserCtx     *types.ParseContext
	execContext   *types.ExecContext
	cfg           *rm.LockConfig
	tx            driver.Tx
	tableName     string
	selectPKSQL   string
	metaData      *types.TableMeta
	savepointName string
	dbType        types.DBType
}

func NewSelectForUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) executor {
	return &selectForUpdateExecutor{
		baseExecutor: baseExecutor{
			hooks: hooks,
		},
		parserCtx:   parserCtx,
		execContext: execContext,
		cfg:         &LockConfig,
		dbType:      execContext.TxCtx.DBType,
	}
}

func (s *selectForUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	s.beforeHooks(ctx, s.execContext)
	defer func() {
		s.afterHooks(ctx, s.execContext)
	}()

	// todo fix IsRequireGlobalLock
	if !tm.IsGlobalTx(ctx) && !s.execContext.IsRequireGlobalLock {
		return f(ctx, s.execContext.Query, s.execContext.NamedValues)
	}

	var (
		result             types.ExecResult
		originalAutoCommit = s.execContext.IsAutoCommit
		err                error
	)

	if s.tableName, err = s.parserCtx.GetTableName(); err != nil {
		return nil, err
	}

	if s.metaData, err = datasource.GetTableCache(s.dbType).GetTableMeta(ctx, s.execContext.DBName, s.tableName); err != nil {
		return nil, err
	}

	// build query primary key sql
	if s.selectPKSQL, err = s.buildSelectPKSQL(s.parserCtx.SelectStmt, s.metaData); err != nil {
		return nil, err
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: s.cfg.RetryTimes,
		MinBackoff: s.cfg.RetryInterval,
		MaxBackoff: s.cfg.RetryInterval,
	})

	for bf.Ongoing() {
		result, err = s.doExecContext(ctx, f)
		if err == nil || errors.Is(err, lockConflictError) {
			break
		}
		bf.Wait()
	}

	if bf.Err() != nil || err != nil {
		if err == nil {
			err = bf.Err()
		}
		// if there is an err in doExecContext, we should rollback first
		if s.savepointName != "" {
			rollbackSQL := s.buildRollbackSQL(s.savepointName)
			if _, rollerr := s.exec(ctx, rollbackSQL, nil, nil); rollerr != nil {
				log.Error("rollback to %s failed, err %s", s.savepointName, rollerr.Error())
				return nil, err
			}
		} else {
			if rollerr := s.tx.Rollback(); rollerr != nil {
				log.Error("rollback failed, err %s", rollerr.Error())
				return nil, err
			}
		}
		return nil, err
	}

	if originalAutoCommit {
		if err = s.tx.Commit(); err != nil {
			return nil, err
		}
		s.execContext.IsAutoCommit = true
	}

	return result, nil
}

func (s *selectForUpdateExecutor) doExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	var (
		now                = time.Now().Unix()
		result             types.ExecResult
		originalAutoCommit = s.execContext.IsAutoCommit
		err                error
	)

	if originalAutoCommit {
		// In order to hold the local db lock during global lock checking
		// set auto commit value to false first if original auto commit was true
		s.execContext.IsAutoCommit = false
		s.tx, err = s.execContext.Conn.Begin()
		if err != nil {
			return nil, err
		}
	} else if s.execContext.IsSupportsSavepoints {
		// In order to release the local db lock when global lock conflict
		// create a save point if original auto commit was false, then use the save point here to release db
		// lock during global lock checking if necessary
		savepointName := s.generateSavepointName(now)
		savepointSQL := s.buildSavepointSQL(savepointName)
		if _, err = s.exec(ctx, savepointSQL, nil, nil); err != nil {
			return nil, err
		}
		s.savepointName = savepointName
	} else {
		return nil, fmt.Errorf("not support savepoint. please check your db version")
	}

	// query primary key values
	var lockKey string
	_, err = s.exec(ctx, s.selectPKSQL, s.execContext.NamedValues, func(rows driver.Rows) {
		lockKey = s.buildLockKey(rows, s.metaData)
	})

	if err != nil {
		return nil, err
	}

	if lockKey == "" {
		return nil, nil
	}

	// execute business SQL, try to get local lock
	result, err = f(ctx, s.execContext.Query, s.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	// check global lock
	lockable, err := datasource.GetDataSourceManager(branch.BranchTypeAT).LockQuery(ctx, rm.LockQueryParam{
		Xid:        s.execContext.TxCtx.XID,
		BranchType: branch.BranchTypeAT,
		ResourceId: s.execContext.TxCtx.ResourceID,
		LockKeys:   lockKey,
	})
	if err != nil {
		return nil, err
	}

	if !lockable {
		return nil, lockConflictError
	}

	return result, nil
}

// buildSelectSQLByUpdate build select sql from update sql
func (s *selectForUpdateExecutor) buildSelectPKSQL(stmt *ast.SelectStmt, meta *types.TableMeta) (string, error) {
	pks := meta.GetPrimaryKeyOnlyName()
	if len(pks) == 0 {
		return "", fmt.Errorf("%s needs to contain the primary key.", meta.TableName)
	}

	var fields []*ast.SelectField
	for _, column := range pks {
		escapedColumn := s.escapeIdentifier(column)
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.CIStr{
						O: escapedColumn,
						L: strings.ToLower(escapedColumn),
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
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := seatabytes.NewByteBuffer([]byte{})
	selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())

	sql = s.adaptSQLSyntax(sql)

	sql = s.escapeAllIdentifiers(sql)

	log.Infof("build select sql by update sourceQuery, sql {}", sql)

	return sql, nil
}

// the string as local key. the local key example(multi pk): "t_user:1_a,2_b"
func (s *selectForUpdateExecutor) buildLockKey(rows driver.Rows, meta *types.TableMeta) string {
	var (
		lockKeys    bytes.Buffer
		idx         int
		columnNames []string
	)
	lockKeys.WriteString(meta.TableName)
	lockKeys.WriteString(":")

	columnNames = meta.GetPrimaryKeyOnlyName()
	sqlRows := util.NewScanRows(rows)
	for sqlRows.Next() {
		ss := s.GetScanSlice(columnNames, meta)
		if err := sqlRows.Scan(ss...); err != nil {
			if err == io.EOF {
				break
			}
			return ""
		}

		if idx > 0 {
			lockKeys.WriteString(",")
		}
		idx++

		for i, value := range ss {
			if i > 0 {
				lockKeys.WriteString("_")
			}

			// if the value is NullInt64 or NullString etc. then call its Value()
			ty := reflect.TypeOf(value)
			if f, ok := ty.MethodByName("Value"); ok {
				res := f.Func.Call([]reflect.Value{reflect.ValueOf(value)})
				if res[1].IsNil() { // res[0]: driver.Value, [1]: error
					lockKeys.WriteString(res[0].Elem().String())
				}
				continue
			}

			// if the value type is *int64, *string etc. then get the true value
			lockKeys.WriteString(fmt.Sprintf("%v", reflect.ValueOf(value).Elem()))
		}
	}
	return lockKeys.String()
}

func (s *selectForUpdateExecutor) exec(ctx context.Context, sql string, nvdargs []driver.NamedValue, f func(rows driver.Rows)) (driver.Rows, error) {
	var (
		querierContext                  driver.QueryerContext
		querier                         driver.Queryer
		queryerCtxExists, queryerExists bool
	)

	if querierContext, queryerCtxExists = s.execContext.Conn.(driver.QueryerContext); !queryerCtxExists {
		if querier, queryerExists = s.execContext.Conn.(driver.Queryer); !queryerExists {
			log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
			return nil, fmt.Errorf("invalid conn")
		}
	}

	rows, err := util.CtxDriverQuery(ctx, querierContext, querier, sql, nvdargs)
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	if err != nil {
		return nil, err
	}

	if f != nil {
		f(rows)
	}

	return nil, nil
}

// generateSavepointName generates savepoint name for different database types
func (s *selectForUpdateExecutor) generateSavepointName(timestamp int64) string {
	switch s.dbType {
	case types.DBTypePostgreSQL:
		// PostgreSQL savepoint names are case-sensitive and should be quoted if they contain special characters
		return fmt.Sprintf("seatago_%d_point", timestamp)
	case types.DBTypeMySQL:
		return fmt.Sprintf("seatago%dpoint", timestamp)
	default:
		return fmt.Sprintf("seatago%dpoint", timestamp)
	}
}

// buildSavepointSQL builds savepoint SQL for different database types
func (s *selectForUpdateExecutor) buildSavepointSQL(savepointName string) string {
	switch s.dbType {
	case types.DBTypePostgreSQL:
		return fmt.Sprintf("SAVEPOINT %s;", s.escapeIdentifier(savepointName))
	case types.DBTypeMySQL:
		return fmt.Sprintf("SAVEPOINT %s;", savepointName)
	default:
		return fmt.Sprintf("SAVEPOINT %s;", savepointName)
	}
}

// buildRollbackSQL builds rollback to savepoint SQL for different database types
func (s *selectForUpdateExecutor) buildRollbackSQL(savepointName string) string {
	switch s.dbType {
	case types.DBTypePostgreSQL:
		return fmt.Sprintf("ROLLBACK TO SAVEPOINT %s;", s.escapeIdentifier(savepointName))
	case types.DBTypeMySQL:
		return fmt.Sprintf("ROLLBACK TO %s;", savepointName)
	default:
		return fmt.Sprintf("ROLLBACK TO %s;", savepointName)
	}
}

// escapeIdentifier escapes identifiers for different database types
func (s *selectForUpdateExecutor) escapeIdentifier(name string) string {
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
		// PostgreSQL stores unquoted identifiers in lowercase
		// If the identifier is all uppercase (likely from meta cache), convert to lowercase
		if name == strings.ToUpper(name) {
			name = strings.ToLower(name)
		}
		return "\"" + name + "\""
	default:
		return name
	}
}

// adaptSQLSyntax adapts SQL syntax for different database types
func (s *selectForUpdateExecutor) adaptSQLSyntax(sql string) string {
	switch s.dbType {
	case types.DBTypePostgreSQL:
		return s.adaptPostgreSQLSyntax(sql)
	case types.DBTypeMySQL:
		return s.adaptMySQLSyntax(sql)
	}
	return sql
}

// adaptPostgreSQLSyntax adapts SQL syntax specifically for PostgreSQL
func (s *selectForUpdateExecutor) adaptPostgreSQLSyntax(sql string) string {
	sql = strings.ReplaceAll(sql, "SQL_NO_CACHE ", "")
	sql = strings.ReplaceAll(sql, "SQL_CACHE ", "")

	sql = s.convertMySQLIdentifiersToPostgreSQL(sql)

	if !strings.Contains(sql, "$") {
		counter := 1
		sql = regexp.MustCompile(`\?`).ReplaceAllStringFunc(sql, func(match string) string {
			result := fmt.Sprintf("$%d", counter)
			counter++
			return result
		})
	}

	return sql
}

// adaptMySQLSyntax adapts SQL syntax specifically for MySQL
func (s *selectForUpdateExecutor) adaptMySQLSyntax(sql string) string {
	sql = s.convertPostgreSQLIdentifiersToMySQL(sql)
	return sql
}

// convertMySQLIdentifiersToPostgreSQL converts MySQL backtick identifiers to PostgreSQL double quotes
func (s *selectForUpdateExecutor) convertMySQLIdentifiersToPostgreSQL(sql string) string {
	sql = regexp.MustCompile("`([^`]+)`").ReplaceAllString(sql, "\"$1\"")
	return sql
}

// convertPostgreSQLIdentifiersToMySQL converts PostgreSQL double quote identifiers to MySQL backticks
func (s *selectForUpdateExecutor) convertPostgreSQLIdentifiersToMySQL(sql string) string {
	sql = regexp.MustCompile("\"([^\"]+)\"").ReplaceAllString(sql, "`$1`")
	return sql
}

func (s *selectForUpdateExecutor) escapeAllIdentifiers(sql string) string {
	sql = regexp.MustCompile(`(?i)\bFROM\s+([a-zA-Z_][a-zA-Z0-9_]*)\b`).ReplaceAllStringFunc(sql, func(match string) string {
		parts := strings.Fields(match)
		if len(parts) == 2 {
			return parts[0] + " " + s.escapeIdentifier(parts[1])
		}
		return match
	})

	sql = regexp.MustCompile(`(?i)\bWHERE\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*([><=!]+)`).ReplaceAllStringFunc(sql, func(match string) string {
		re := regexp.MustCompile(`(?i)(\bWHERE\s+)([a-zA-Z_][a-zA-Z0-9_]*)(\s*[><=!]+)`)
		return re.ReplaceAllString(match, "${1}"+s.escapeIdentifier("${2}")+"${3}")
	})

	return sql
}
