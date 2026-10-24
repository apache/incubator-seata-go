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
	"github.com/arana-db/parser/model"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type multiDeleteExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

func (m *multiDeleteExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if err := m.beforeHooks(ctx, m.execContext); err != nil {
		return nil, err
	}
	defer func() {
		m.afterHooks(ctx, m.execContext)
	}()
	if len(m.parserCtx.MultiStmt) == 0 {
		return nil, fmt.Errorf("aggregate delete has no statements")
	}
	if err := m.resolveTableMetaKey(ctx, m.execContext, m.parserCtx.MultiStmt[0]); err != nil {
		return nil, err
	}

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

	for _, image := range beforeImage {
		image.TableMetaKey = m.execContext.TableMetaKey
	}
	for _, image := range afterImage {
		image.TableMetaKey = m.execContext.TableMetaKey
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

	rowsi, err = util.CtxDriverQueryWithPrepareFallback(ctx, m.execContext.Conn, selectSQL, args)
	if err != nil {
		log.Errorf("aggregate delete image query failed: %+v", err)
		return nil, err
	}
	defer func() {
		if rowsi == nil {
			return
		}

		if closeErr := rowsi.Close(); closeErr != nil {
			log.Errorf("rows close fail,err: %v", closeErr)
		}
	}()

	metaData, err := m.getTableMeta(ctx, m.execContext, m.parserCtx.MultiStmt[0])
	if err != nil {
		return nil, err
	}
	image, err = m.buildRecordImages(rowsi, metaData, types.SQLTypeDelete, types.DBTypeMySQL)
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
	metaData, err := m.getTableMeta(ctx, m.execContext, m.parserCtx.MultiStmt[0])
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

		newParams := m.buildSelectArgs(&ast.SelectStmt{
			Where:      deleteParser.Where,
			From:       deleteParser.TableRefs,
			Limit:      deleteParser.Limit,
			OrderBy:    deleteParser.Order,
			TableHints: deleteParser.TableHints,
		}, m.execContext.NamedValues)
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
			if m.execContext.TableMetaKey == nil {
				return parser.GetTableName()
			}
			if parser.DeleteStmt == nil || parser.DeleteStmt.TableRefs == nil || parser.DeleteStmt.TableRefs.TableRefs == nil {
				return "", fmt.Errorf("multi delete sql has no table reference")
			}
			refs := *parser.DeleteStmt.TableRefs
			join := *refs.TableRefs
			source, ok := join.Left.(*ast.TableSource)
			if !ok || join.Right != nil {
				return "", fmt.Errorf("multi delete sql requires a single table source")
			}
			table, ok := source.Source.(*ast.TableName)
			if !ok {
				return "", fmt.Errorf("multi delete sql requires a table name")
			}

			// Qualify only the table identity, retaining partitions, aliases and hints
			// without changing the AST used to execute the original DELETE statements.
			qualifiedTable := *table
			qualifiedTable.Schema = model.NewCIStr(m.execContext.TableMetaKey.DBName)
			qualifiedTable.Name = model.NewCIStr(m.execContext.TableMetaKey.TableName)
			qualifiedSource := *source
			qualifiedSource.Source = &qualifiedTable
			join.Left = &qualifiedSource
			refs.TableRefs = &join
			var buffer bytes.Buffer
			if err := refs.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase|format.RestoreNameBackQuotes, &buffer)); err != nil {
				return "", err
			}
			return buffer.String(), nil
		}
	}
	return "", fmt.Errorf("multi delete sql has no table name")
}
