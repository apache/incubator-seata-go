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
	"reflect"
	"time"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/util/backoff"
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

	if !s.execContext.TxCtx.IsATMode() && !s.execContext.IsRequireGlobalLock {
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

	if err = s.resolveTableMetaKey(ctx, s.execContext, s.parserCtx); err != nil {
		return nil, err
	}
	if s.metaData, err = s.getTableMeta(ctx, s.execContext, s.parserCtx); err != nil {
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
			if _, rollerr := util.CtxDriverExecWithPrepareFallback(ctx, s.execContext.Conn, "ROLLBACK TO SAVEPOINT "+s.savepointName, nil); rollerr != nil {
				log.Error("rollback to %s failed, err %s", s.savepointName, rollerr.Error())
				return nil, err
			}
		} else if s.tx != nil {
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
		savepointName := fmt.Sprintf("seatago%dpoint", now)
		log.Infof("calling exec for savepoint: %s", savepointName)
		if _, err = util.CtxDriverExecWithPrepareFallback(ctx, s.execContext.Conn, "SAVEPOINT "+savepointName, nil); err != nil {
			log.Errorf("savepoint exec failed: %v", err)
			return nil, err
		}
		log.Infof("savepoint created successfully")
		s.savepointName = savepointName
	} else {
		return nil, fmt.Errorf("not support savepoint. please check your db version")
	}

	// query primary key values
	log.Infof("querying primary key values with SQL: %s", s.selectPKSQL)
	keyRows, err := util.CtxDriverQueryWithPrepareFallback(ctx, s.execContext.Conn, s.selectPKSQL, s.execContext.NamedValues)
	if err != nil {
		return nil, err
	}
	lockKey, err := s.buildLockKey(keyRows, s.metaData)
	err = errors.Join(err, keyRows.Close())
	log.Infof("primary key query completed, lockKey: %s, err: %v", lockKey, err)

	if err != nil {
		return nil, err
	}

	result, err := f(ctx, s.execContext.Query, s.execContext.NamedValues)
	if err != nil || lockKey == "" {
		return result, err
	}

	log.Infof("checking global lock for lockKey: %s", lockKey)
	lockable, err := datasource.GetDataSourceManager(branch.BranchTypeAT).LockQuery(ctx, rm.LockQueryParam{
		Xid:        s.execContext.TxCtx.XID,
		BranchType: branch.BranchTypeAT,
		ResourceId: s.execContext.TxCtx.ResourceID,
		LockKeys:   lockKey,
	})
	if err == nil && lockable {
		return result, nil
	}
	if err == nil {
		err = lockConflictError
	}
	// A query result must be closed before rolling back to the savepoint.
	if result.GetResult() == nil {
		err = errors.Join(err, result.GetRows().Close())
	}
	return nil, err
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

	dbType := types.DBTypeMySQL
	if s.execContext != nil {
		dbType = effectiveDBType(s.execContext.DBType)
	}
	if dbType == types.DBTypePostgreSQL {
		parsed, err := parser.ParsePostgreSQLForImage(stmt.Text())
		if err != nil {
			return "", err
		}
		stmt = parsed.SelectStmt
		if stmt == nil {
			return "", fmt.Errorf("postgres lock query source is not a select statement")
		}
	}
	from := stmt.From
	if dbType == types.DBTypePostgreSQL {
		var err error
		from, err = postgresAuxiliaryQueryTable(from, s.execContext.TableMetaKey)
		if err != nil {
			return "", err
		}
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           from,
		Where:          stmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		OrderBy:        stmt.OrderBy,
		Limit:          stmt.Limit,
		TableHints:     stmt.TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	sql, err := s.restoreImageSelectSQL(&selStmt, dbType)
	if err != nil {
		return "", err
	}
	log.Infof("build select sql by update sourceQuery, sql {}", sql)

	return sql, nil
}

func (s *selectForUpdateExecutor) buildLockKey(rows driver.Rows, meta *types.TableMeta) (string, error) {
	var lockKeys bytes.Buffer
	columnNames := meta.GetPrimaryKeyOnlyName()
	sqlRows := util.NewScanRows(rows)
	for sqlRows.Next() {
		values := s.GetScanSlice(columnNames, meta)
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
