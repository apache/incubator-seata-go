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
	"context"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

// deleteExecutor execute delete SQL
type deleteExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

// NewDeleteExecutor get delete executor
func NewDeleteExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &deleteExecutor{
		parserCtx:   parserCtx,
		execContext: execContent,
		baseExecutor: baseExecutor{
			hooks: hooks,
		},
	}
}

// ExecContext exec SQL, and generate before image and after image
func (d *deleteExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	d.beforeHooks(ctx, d.execContext)
	defer func() {
		d.afterHooks(ctx, d.execContext)
	}()

	beforeImage, err := d.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, d.execContext.Query, d.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := d.afterImage(ctx)
	if err != nil {
		return nil, err
	}

	d.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	d.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

// beforeImage build before image
func (d *deleteExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	selectSQL, selectArgs, err := d.buildBeforeImageSQL(d.execContext.Query, d.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	var rowsi driver.Rows
	queryerCtx, ok := d.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = d.execContext.Conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("execute before-image select sql failed: %+v, sql: %s", err, selectSQL)
			return nil, err
		}
	} else {
		log.Errorf("database connection does not support QueryerContext/Queryer")
		return nil, fmt.Errorf("invalid database connection")
	}

	tableName, _ := d.parserCtx.GetTableName()
	dbType := d.execContext.TxCtx.DBType
	metaData, err := datasource.GetTableCache(dbType).GetTableMeta(ctx, d.execContext.DBName, tableName)
	if err != nil {
		log.Errorf("get table meta failed: %+v, dbType: %s, table: %s", err, dbType, tableName)
		return nil, err
	}

	image, err := d.buildRecordImages(ctx, d.execContext, rowsi, metaData, types.SQLTypeDelete)
	if err != nil {
		log.Errorf("build before-image failed: %+v", err)
		return nil, err
	}
	image.SQLType = types.SQLTypeDelete
	image.TableMeta = metaData

	lockKey := d.buildLockKey(image, *metaData)
	d.execContext.TxCtx.LockKeys[lockKey] = struct{}{}

	return image, nil
}

func (d *deleteExecutor) buildBeforeImageSQL(query string, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	if d.parserCtx == nil {
		log.Errorf("parser context is nil")
		return "", nil, fmt.Errorf("parser context is nil")
	}

	if d.parserCtx.DeleteStmt == nil && d.parserCtx.AuxtenDeleteStmt == nil {
		log.Errorf("invalid delete statement: %s", query)
		return "", nil, fmt.Errorf("invalid delete sql")
	}

	var selectSQL string
	var selectArgs []driver.NamedValue
	dbType := d.execContext.TxCtx.DBType

	if dbType == types.DBTypeMySQL && d.parserCtx.DeleteStmt != nil {
		// MySQL
		selStmt := &ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{
				SQLCache: true,
			},
			From:  d.parserCtx.DeleteStmt.TableRefs,
			Where: d.parserCtx.DeleteStmt.Where,
			Fields: &ast.FieldList{
				Fields: []*ast.SelectField{{
					WildCard: &ast.WildCardField{},
				}},
			},
			OrderBy: d.parserCtx.DeleteStmt.Order,
			Limit:   d.parserCtx.DeleteStmt.Limit,
			LockInfo: &ast.SelectLockInfo{
				LockType: ast.SelectLockForUpdate,
			},
		}

		b := bytes.NewByteBuffer([]byte{})
		restoreCtx := format.NewRestoreCtx(format.RestoreKeyWordUppercase, b)
		if err := selStmt.Restore(restoreCtx); err != nil {
			log.Errorf("restore select sql failed: %+v", err)
			return "", nil, err
		}

		selectSQL = string(b.Bytes())

		re := regexp.MustCompile(`(_UTF8MB4)(\w+)`)
		selectSQL = re.ReplaceAllString(selectSQL, `${1}'${2}'`)

		if !strings.Contains(selectSQL, "SQL_NO_CACHE") {
			selectSQL = strings.Replace(selectSQL, "SELECT", "SELECT SQL_NO_CACHE", 1)
		}

		selectArgs = d.buildSelectArgs(selStmt, args)
	} else if dbType == types.DBTypePostgreSQL && d.parserCtx.AuxtenDeleteStmt != nil {
		deleteStmt := d.parserCtx.AuxtenDeleteStmt

		selectSQL = "SELECT * FROM "

		if deleteStmt.Table != nil {
			ctx := tree.NewFmtCtx(tree.FmtSimple)
			deleteStmt.Table.Format(ctx)
			selectSQL += ctx.CloseAndGetString()
		}

		if deleteStmt.Where != nil {
			ctx := tree.NewFmtCtx(tree.FmtSimple)
			deleteStmt.Where.Format(ctx)
			whereClause := ctx.CloseAndGetString()
			if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(whereClause)), "WHERE") {
				selectSQL += " WHERE " + whereClause
			} else {
				selectSQL += " " + whereClause
			}
		}

		if len(deleteStmt.OrderBy) > 0 {
			ctx := tree.NewFmtCtx(tree.FmtSimple)
			deleteStmt.OrderBy.Format(ctx)
			orderClause := ctx.CloseAndGetString()
			if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(orderClause)), "ORDER BY") {
				selectSQL += " ORDER BY " + orderClause
			} else {
				selectSQL += " " + orderClause
			}
		}

		if deleteStmt.Limit != nil {
			ctx := tree.NewFmtCtx(tree.FmtSimple)
			deleteStmt.Limit.Format(ctx)
			limitClause := ctx.CloseAndGetString()
			if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(limitClause)), "LIMIT") {
				selectSQL += " LIMIT " + limitClause
			} else {
				selectSQL += " " + limitClause
			}
		}

		selectSQL += " FOR UPDATE"

		selectArgs = args
	} else {
		log.Errorf("unsupported database type or missing delete statement, dbType: %v", dbType)
		return "", nil, fmt.Errorf("invalid delete sql")
	}

	log.Infof("built before-image select sql: %s", selectSQL)
	return selectSQL, selectArgs, nil
}

// afterImage build after image
func (d *deleteExecutor) afterImage(ctx context.Context) (*types.RecordImage, error) {
	tableName, _ := d.parserCtx.GetTableName()
	dbType := d.execContext.TxCtx.DBType
	metaData, err := datasource.GetTableCache(dbType).GetTableMeta(ctx, d.execContext.DBName, tableName)
	if err != nil {
		log.Errorf("get table meta for after-image failed: %+v", err)
		return nil, err
	}

	return types.NewEmptyRecordImage(metaData, types.SQLTypeDelete), nil
}
