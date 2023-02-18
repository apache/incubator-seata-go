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
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"

	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/util/backoff"
	seatabytes "github.com/seata/seata-go/pkg/util/bytes"
	"github.com/seata/seata-go/pkg/util/log"
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
	s.beforeHooks(ctx, s.execContext)
	defer func() {
		s.afterHooks(ctx, s.execContext)
	}()

	if !s.execContext.IsInGlobalTransaction && !s.execContext.IsRequireGlobalLock {
		return f(ctx, s.execContext.Query, s.execContext.NamedValues)
	}

	var (
		result             types.ExecResult
		originalAutoCommit = s.execContext.IsAutoCommit
		err                error
	)

	if s.tableName, err = s.execContext.ParseContext.GetTableName(); err != nil {
		return nil, err
	}

	if s.metaData, err = datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, s.execContext.DBName, s.tableName); err != nil {
		return nil, err
	}

	// build query primary key sql
	if s.selectPKSQL, err = s.buildSelectPKSQL(s.execContext.ParseContext.SelectStmt, s.metaData); err != nil {
		return nil, err
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: s.cfg.RetryTimes,
		MinBackoff: s.cfg.RetryInterval,
		MaxBackoff: s.cfg.RetryInterval,
	})

	for bf.Ongoing() {
		if result, err = s.doExecContext(ctx, f); err == nil {
			break
		}

		// if there is an err in doExecContext, we should rollback first
		if s.savepointName != "" {
			if _, err := s.exec(fmt.Sprintf("rollback to %s;", s.savepointName), nil); err != nil {
				log.Error(err)
				return nil, err
			}
		} else {
			if err = s.tx.Rollback(); err != nil {
				return nil, err
			}
		}

		bf.Wait()
	}

	if bf.Err() != nil {
		lastErr := fmt.Errorf("lastErr %v, backoff error: %v", err, bf.Err())
		log.Warnf("select for update executor failed: %v", lastErr)
		return nil, lastErr
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
		if _, err = s.exec(fmt.Sprintf("savepoint %d;", now), nil); err != nil {
			return nil, err
		}
		s.savepointName = strconv.FormatInt(now, 10)
	} else {
		return nil, fmt.Errorf("not support savepoint. please check your db version")
	}

	// execute business SQL, try to get local lock
	result, err = f(ctx, s.execContext.Query, s.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	// query primary key values
	var lockKey string
	if _, err = s.exec(s.selectPKSQL, func(rows driver.Rows) {
		lockKey = s.buildLockKey(rows, s.metaData)
	}); err != nil {
		return nil, err
	}
	if lockKey == "" {
		return nil, nil
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
		return nil, fmt.Errorf("get lock failed, lockKey: %v", lockKey)
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
	sql := string(b.Bytes())
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

func (s *selectForUpdateExecutor) exec(sql string, f func(rows driver.Rows)) (driver.Rows, error) {
	var (
		querierContext driver.QueryerContext
		querier        driver.Queryer
		ok             bool
	)
	if querierContext, ok = s.execContext.Conn.(driver.QueryerContext); !ok {
		err := fmt.Sprintf("invalid conn, can't convert %v to driver.QueryerContext", s.execContext.Conn)
		if querier, ok = s.execContext.Conn.(driver.Queryer); !ok {
			err = err + fmt.Sprintf(", also can't convert %v to drvier.Queryer", s.execContext.Conn)
			return nil, fmt.Errorf(err)
		}
	}

	rows, err := util.CtxDriverQuery(context.TODO(), querierContext, querier, sql, nil)
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
