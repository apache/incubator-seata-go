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
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/util/log"
)

type multiDeleteExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

func (m *multiDeleteExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	m.beforeHooks(ctx, m.execContext)
	defer func() {
		m.afterHooks(ctx, m.execContext)
	}()

	beforeImage, err := m.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, m.execContext.Query, m.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := m.afterImage(ctx)
	if err != nil {
		return nil, err
	}

	m.execContext.TxCtx.RoundImages.AppendBeofreImages(beforeImage)
	m.execContext.TxCtx.RoundImages.AppendAfterImages(afterImage)
	return res, nil
}

type multiDelete struct {
	sql   string
	clear bool
}

//NewMultiDeleteExecutor get multiDelete executor
func NewMultiDeleteExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) *multiDeleteExecutor {
	return &multiDeleteExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

func (m *multiDeleteExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	multiQuery, args, err := m.buildBeforeImageSQL()
	if err != nil {
		return nil, err
	}
	var (
		rowsi   driver.Rows
		image   *types.RecordImage
		records []*types.RecordImage
	)

	queryerCtx, ok := m.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = m.execContext.Conn.(driver.Queryer)
	}
	if !ok {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}
	for i, sql := range multiQuery {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, sql, args)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
		tableName := m.parserCtx.MultiStmt[i].DeleteStmt.
			TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
		metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, m.execContext.DBName, tableName)
		if err != nil {
			return nil, err
		}
		image, err = m.buildRecordImages(rowsi, metaData, types.SQLTypeDelete)
		if err != nil {
			log.Errorf("record images : %+v", err)
			return nil, err
		}
		records = append(records, image)
		lockKey := m.buildLockKey(image, *metaData)
		m.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	}
	return records, err
}

func (m *multiDeleteExecutor) afterImage(ctx context.Context) ([]*types.RecordImage, error) {
	tableName, _ := m.parserCtx.GetTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, m.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	image := types.NewEmptyRecordImage(metaData, types.SQLTypeDelete)
	return []*types.RecordImage{image}, nil
}

func (m *multiDeleteExecutor) buildBeforeImageSQL() ([]string, []driver.NamedValue, error) {
	var (
		err        error
		buf, param bytes.Buffer
		p          *types.ParseContext
		tableName  string
		args       = m.execContext.NamedValues
		multiQuery = strings.Split(m.execContext.Query, ";")
		tables     = make(map[string]multiDelete, len(multiQuery))
	)

	ps, err := parser.DoParser(m.execContext.Query)
	if err != nil {
		return nil, nil, err
	}
	for _, p = range ps.MultiStmt {
		tableName = p.DeleteStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
		v, ok := tables[tableName]
		if ok && v.clear {
			continue
		}
		buf.WriteString("delete from ")
		buf.WriteString(tableName)
		if p.DeleteStmt.Where == nil {
			tables[tableName] = multiDelete{sql: buf.String(), clear: true}
			buf.Reset()
			continue
		} else {
			buf.WriteString(" where ")
		}

		_ = p.DeleteStmt.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &param))
		v, ok = tables[tableName]
		if ok {
			buf.Reset()
			buf.WriteString(v.sql)
			buf.WriteString(" or ")
		}

		buf.Write(param.Bytes())
		tables[tableName] = multiDelete{sql: buf.String()}

		buf.Reset()
		param.Reset()
	}

	var (
		items   = make([]string, 0, len(tables))
		values  = make([]driver.NamedValue, 0, len(tables))
		selStmt = ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{},
			From:           p.DeleteStmt.TableRefs,
			Where:          p.DeleteStmt.Where,
			Fields:         &ast.FieldList{Fields: []*ast.SelectField{{WildCard: &ast.WildCardField{}}}},
			OrderBy:        p.DeleteStmt.Order,
			TableHints:     p.DeleteStmt.TableHints,
			LockInfo:       &ast.SelectLockInfo{LockType: ast.SelectLockForUpdate},
		}
	)
	for _, table := range tables {
		p, _ = parser.DoParser(table.sql)

		selStmt.From = p.DeleteStmt.TableRefs
		selStmt.Where = p.DeleteStmt.Where
		_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &buf))
		items = append(items, buf.String())
		buf.Reset()
		if table.clear {
			values = append(values, m.buildSelectArgs(&selStmt, nil)...)
		} else {
			values = append(values, m.buildSelectArgs(&selStmt, args)...)
		}
	}
	return items, values, nil
}
