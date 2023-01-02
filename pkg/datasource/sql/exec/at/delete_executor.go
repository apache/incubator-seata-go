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

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/util/bytes"
	"github.com/seata/seata-go/pkg/util/log"
)

// deleteExecutor execute delete SQL
type deleteExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContent *types.ExecContext
}

// NewDeleteExecutor get delete executor
func NewDeleteExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &deleteExecutor{parserCtx: parserCtx, execContent: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

// ExecContext exec SQL, and generate before image and after image
func (d deleteExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	d.beforeHooks(ctx, d.execContent)
	defer func() {
		d.afterHooks(ctx, d.execContent)
	}()

	beforeImage, err := d.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, d.execContent.Query, d.execContent.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := d.afterImage(ctx)
	if err != nil {
		return nil, err
	}

	d.execContent.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	d.execContent.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

// beforeImage build before image
func (d *deleteExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	selectSQL, selectArgs, err := d.buildBeforeImageSQL(d.execContent.Query, d.execContent.NamedValues)
	if err != nil {
		return nil, err
	}

	var rowsi driver.Rows
	queryerCtx, ok := d.execContent.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = d.execContent.Conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	tableName, _ := d.parserCtx.GteTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, d.execContent.DBName, tableName)
	if err != nil {
		return nil, err
	}

	image, err := d.buildRecordImages(rowsi, metaData)
	if err != nil {
		return nil, err
	}
	image.SQLType = types.SQLTypeDelete
	image.TableMeta = metaData

	lockKey := d.buildLockKey(image, *metaData)
	d.execContent.TxCtx.LockKeys[lockKey] = struct{}{}

	return image, nil
}

// buildBeforeImageSQL build delete sql from delete sql
func (d *deleteExecutor) buildBeforeImageSQL(query string, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	p, err := parser.DoParser(query)
	if err != nil {
		return "", nil, err
	}

	if p.DeleteStmt == nil {
		log.Errorf("invalid delete stmt")
		return "", nil, fmt.Errorf("invalid delete stmt")
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           p.DeleteStmt.TableRefs,
		Where:          p.DeleteStmt.Where,
		Fields:         &ast.FieldList{Fields: []*ast.SelectField{{WildCard: &ast.WildCardField{}}}},
		OrderBy:        p.DeleteStmt.Order,
		Limit:          p.DeleteStmt.Limit,
		TableHints:     p.DeleteStmt.TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := bytes.NewByteBuffer([]byte{})
	_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by delete sourceQuery, sql {%s}", sql)

	return sql, d.buildSelectArgs(&selStmt, args), nil
}

// afterImage build after image
func (d *deleteExecutor) afterImage(ctx context.Context) (*types.RecordImage, error) {
	tableName, _ := d.parserCtx.GteTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, d.execContent.DBName, tableName)
	if err != nil {
		return nil, err
	}
	return types.NewEmptyRecordImage(metaData, types.SQLTypeDelete), nil
}
