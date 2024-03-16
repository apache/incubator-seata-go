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
	"context"
	"database/sql/driver"
	"fmt"

	"seata.apache.org/seata-go/pkg/datasource/sql/undo"

	"seata.apache.org/seata-go/pkg/datasource/sql/parser"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

type MySQLDeleteUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetMySQLDeleteUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLDeleteUndoLogBuilder{}
}

func (u *MySQLDeleteUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	vals := execCtx.Values
	if vals == nil {
		for n, param := range execCtx.NamedValues {
			vals[n] = param.Value
		}
	}
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(execCtx.Query, vals)
	if err != nil {
		return nil, err
	}

	stmt, err := execCtx.Conn.Prepare(selectSQL)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}

	tableName := execCtx.ParseContext.DeleteStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]

	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey(rows, metaData)
	execCtx.TxCtx.LockKeys[lockKey] = struct{}{}

	return []*types.RecordImage{image}, nil
}

func (u *MySQLDeleteUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforImages []*types.RecordImage) ([]*types.RecordImage, error) {
	return nil, nil
}

// buildBeforeImageSQL build delete sql from delete sql
func (u *MySQLDeleteUndoLogBuilder) buildBeforeImageSQL(query string, args []driver.Value) (string, []driver.Value, error) {
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

	return sql, u.buildSelectArgs(&selStmt, args), nil
}

func (u *MySQLDeleteUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.DeleteExecutor
}
