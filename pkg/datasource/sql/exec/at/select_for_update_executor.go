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
	"strings"
	"time"

	"seata.apache.org/seata-go/v2/pkg/tm"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/util/backoff"
	seatabytes "seata.apache.org/seata-go/v2/pkg/util/bytes"
	"seata.apache.org/seata-go/v2/pkg/util/log"
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
}

func NewSelectForUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) executor {
	return &selectForUpdateExecutor{
		baseExecutor: baseExecutor{
			hooks: hooks,
		},
		parserCtx:   parserCtx,
		execContext: execContext,
		cfg:         &LockConfig,
	}
}

func (s *selectForUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if err := s.beforeHooks(ctx, s.execContext); err != nil {
		return nil, err
	}
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

	dbType := effectiveDBType(s.execContext.DBType)
	if s.metaData, err = datasource.GetTableCache(dbType).GetTableMeta(ctx, s.execContext.DBName, s.tableName); err != nil {
		return nil, err
	}

	// build query primary key sql
	if s.selectPKSQL, err = s.buildSelectPKSQL(s.parserCtx.SelectStmt, s.metaData); err != nil {
		return nil, err
	}
	log.Infof("selectPKSQL built successfully")

	log.Infof("creating backoff config")
	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: s.cfg.RetryTimes,
		MinBackoff: s.cfg.RetryInterval,
		MaxBackoff: s.cfg.RetryInterval,
	})
	log.Infof("backoff created, entering retry loop")

	for bf.Ongoing() {
		log.Infof("calling doExecContext")
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
			if _, rollerr := s.exec(ctx, fmt.Sprintf("rollback to %s;", s.savepointName), nil, nil); rollerr != nil {
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

	log.Infof("doExecContext started, originalAutoCommit: %v", originalAutoCommit)

	if originalAutoCommit {
		// In order to hold the local db lock during global lock checking
		// set auto commit value to false first if original auto commit was true
		s.execContext.IsAutoCommit = false
		log.Infof("calling Begin() on connection")
		s.tx, err = s.execContext.Conn.Begin()
		log.Infof("Begin() returned, err: %v", err)
		if err != nil {
			return nil, err
		}
	} else if s.execContext.IsSupportsSavepoints {
		// In order to release the local db lock when global lock conflict
		// create a save point if original auto commit was false, then use the save point here to release db
		// lock during global lock checking if necessary
		log.Infof("originalAutoCommit is false, creating savepoint")
		savepointName := fmt.Sprintf("seatago%dpoint;", now)
		log.Infof("calling exec for savepoint: %s", savepointName)
		if _, err = s.exec(ctx, fmt.Sprintf("savepoint %s;", savepointName), nil, nil); err != nil {
			log.Errorf("savepoint exec failed: %v", err)
			return nil, err
		}
		log.Infof("savepoint created successfully")
		s.savepointName = savepointName
	} else {
		return nil, fmt.Errorf("not support savepoint. please check your db version")
	}

	// query primary key values
	var lockKey string
	log.Infof("querying primary key values with SQL: %s", s.selectPKSQL)
	_, err = s.exec(ctx, s.selectPKSQL, s.execContext.NamedValues, func(rows driver.Rows) {
		lockKey = s.buildLockKey(rows, s.metaData)
	})
	log.Infof("primary key query completed, lockKey: %s, err: %v", lockKey, err)

	if err != nil {
		return nil, err
	}

	if lockKey == "" {
		return nil, nil
	}

	// execute business SQL, try to get local lock
	log.Infof("executing business SQL, query: %s", s.execContext.Query)
	result, err = f(ctx, s.execContext.Query, s.execContext.NamedValues)
	log.Infof("business SQL executed, err: %v", err)
	if err != nil {
		return nil, err
	}

	// check global lock
	log.Infof("checking global lock for lockKey: %s", lockKey)
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
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.CIStr{
						O: column,
						L: column,
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
	dbType := types.DBTypeMySQL
	if s.execContext != nil {
		dbType = effectiveDBType(s.execContext.DBType)
	}
	sql := s.normalizeGeneratedSQL(string(b.Bytes()), dbType)
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
		rows                            driver.Rows
		err                             error
	)

	log.Debugf("exec called with sql: %s", sql)

	// Try direct query first
	if querierContext, queryerCtxExists = s.execContext.Conn.(driver.QueryerContext); queryerCtxExists ||
		func() bool { querier, queryerExists = s.execContext.Conn.(driver.Queryer); return queryerExists }() {
		log.Debugf("attempting direct query")
		rows, err = util.CtxDriverQuery(ctx, querierContext, querier, sql, nvdargs)
		log.Debugf("direct query returned, err: %v", err)
		if err == nil {
			if f != nil {
				defer func() {
					if rows != nil {
						_ = rows.Close()
					}
				}()
				f(rows)
				return nil, nil
			}
			return rows, nil
		}
		// If not skip-fast-path error, return the error
		if !strings.Contains(err.Error(), "skip fast-path") {
			return nil, err
		}
		// skip-fast-path error, fallback to prepared statement
		log.Debugf("direct query not supported, falling back to prepared statement")
	}

	// Fallback: use PrepareContext + QueryContext
	log.Debugf("preparing statement for sql: %s", sql)
	stmt, err := s.execContext.Conn.Prepare(sql)
	if err != nil {
		log.Errorf("prepare statement failed: %+v", err)
		return nil, err
	}

	log.Debugf("executing prepared statement query")
	if stmtQueryCtx, ok := stmt.(driver.StmtQueryContext); ok {
		rows, err = stmtQueryCtx.QueryContext(ctx, nvdargs)
	} else {
		dargs, convErr := namedValuesToValues(nvdargs)
		if convErr != nil {
			stmt.Close()
			return nil, convErr
		}
		rows, err = stmt.Query(dargs)
	}
	log.Debugf("prepared statement query completed, err: %v", err)

	if err != nil {
		stmt.Close()
		return nil, err
	}

	if f != nil {
		defer func() {
			if rows != nil {
				_ = rows.Close()
			}
			stmt.Close()
		}()
		f(rows)
		return nil, nil
	}

	return util.NewRowsWithStmt(rows, stmt), nil
}
