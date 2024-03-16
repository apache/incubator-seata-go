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

package builder

import (
	"bytes"
	"context"
	"database/sql/driver"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/log"
)

type multiDelete struct {
	sql   string
	clear bool
}

type MySQLMultiDeleteUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetMySQLMultiDeleteUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLMultiDeleteUndoLogBuilder{BasicUndoLogBuilder: BasicUndoLogBuilder{}}
}

func (u *MySQLMultiDeleteUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	deletes := strings.Split(execCtx.Query, ";")
	if len(deletes) == 1 {
		return GetMySQLDeleteUndoLogBuilder().BeforeImage(ctx, execCtx)
	}

	values := make([]driver.Value, 0, len(execCtx.NamedValues)*2)
	if execCtx.Values == nil {
		for n, param := range execCtx.NamedValues {
			values[n] = param.Value
		}
	}

	multiQuery, args, err := u.buildBeforeImageSQL(deletes, values)
	if err != nil {
		return nil, err
	}

	var (
		stmt driver.Stmt
		rows driver.Rows

		record  *types.RecordImage
		records []*types.RecordImage

		meDataMap = execCtx.MetaDataMap[execCtx.ParseContext.DeleteStmt.
				TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O]
	)

	for _, sql := range multiQuery {
		stmt, err = execCtx.Conn.Prepare(sql)
		if err != nil {
			log.Errorf("build prepare stmt: %+v", err)
			return nil, err
		}

		rows, err = stmt.Query(args)
		if err != nil {
			log.Errorf("stmt query: %+v", err)
			return nil, err
		}

		record, err = u.buildRecordImages(rows, &meDataMap)
		if err != nil {
			log.Errorf("record images : %+v", err)
			return nil, err
		}
		records = append(records, record)

		lockKey := u.buildLockKey(rows, meDataMap)
		execCtx.TxCtx.LockKeys[lockKey] = struct{}{}
	}

	return records, nil
}

func (u *MySQLMultiDeleteUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	return nil, nil
}

// buildBeforeImageSQL build delete sql from delete sql
func (u *MySQLMultiDeleteUndoLogBuilder) buildBeforeImageSQL(multiQuery []string, args []driver.Value) ([]string, []driver.Value, error) {
	var (
		err        error
		buf, param bytes.Buffer
		p          *types.ParseContext
		tableName  string
		tables     = make(map[string]multiDelete, len(multiQuery))
	)

	for _, query := range multiQuery {
		p, err = parser.DoParser(query)
		if err != nil {
			return nil, nil, err
		}

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
		values  = make([]driver.Value, 0, len(tables))
		selStmt = ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{},
			From:           p.DeleteStmt.TableRefs,
			Where:          p.DeleteStmt.Where,
			Fields:         &ast.FieldList{Fields: []*ast.SelectField{{WildCard: &ast.WildCardField{}}}},
			OrderBy:        p.DeleteStmt.Order,
			Limit:          p.DeleteStmt.Limit,
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
			values = append(values, u.buildSelectArgs(&selStmt, nil)...)
		} else {
			values = append(values, u.buildSelectArgs(&selStmt, args)...)
		}
	}

	return items, values, nil
}

func (u *MySQLMultiDeleteUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.MultiDeleteExecutor
}
