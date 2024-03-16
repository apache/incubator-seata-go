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

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/log"
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

// NewMultiDeleteExecutor get multiDelete executor
func NewMultiDeleteExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) *multiDeleteExecutor {
	return &multiDeleteExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

func (m *multiDeleteExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	selectSQL, args, err := m.buildBeforeImageSQL()
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

	rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, args)
	defer func() {
		if rowsi != nil {
			rowsi.Close()
		}
	}()
	if err != nil {
		log.Errorf("ctx driver query: %+v", err)
		return nil, err
	}

	tableName, err := m.getFromTableInSQL()
	if err != nil {
		return nil, err
	}
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

func (m *multiDeleteExecutor) buildBeforeImageSQL() (string, []driver.NamedValue, error) {
	tableName, err := m.getFromTableInSQL()
	if err != nil {
		return "", nil, err
	}

	var (
		// todo optimize replace * by use columns
		selectSQL         = "SELECT SQL_NO_CACHE * FROM " + tableName
		params            []driver.NamedValue
		whereCondition    string
		hasWhereCondition = true
	)

	for _, parser := range m.parserCtx.MultiStmt {
		deleteParser := parser.DeleteStmt
		if deleteParser == nil {
			continue
		}

		if deleteParser.Limit != nil {
			return "", nil, fmt.Errorf("Multi delete SQL with limit condition is not support yet!")
		}
		if deleteParser.Order != nil {
			return "", nil, fmt.Errorf("Multi delete SQL with orderBy condition is not support yet!")
		}
		if deleteParser.Where == nil || !hasWhereCondition {
			hasWhereCondition = false
			continue
		}

		var whereBuffer bytes.Buffer
		if err = deleteParser.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &whereBuffer)); err != nil {
			return "", nil, err
		}

		if whereCondition != "" {
			whereCondition += " OR "
		}
		whereCondition += fmt.Sprintf("(%s)", string(whereBuffer.Bytes()))

		newParams := m.buildSelectArgs(&ast.SelectStmt{Where: parser.DeleteStmt.Where}, m.execContext.NamedValues)
		params = append(params, newParams...)
	}

	if hasWhereCondition {
		selectSQL += " WHERE " + whereCondition
	} else {
		params = []driver.NamedValue{}
	}
	selectSQL += " FOR UPDATE"

	return selectSQL, params, nil
}

func (m *multiDeleteExecutor) getFromTableInSQL() (string, error) {
	for _, parser := range m.parserCtx.MultiStmt {
		if parser != nil {
			return parser.GetTableName()
		}
	}
	return "", fmt.Errorf("multi delete sql has no table name")
}
