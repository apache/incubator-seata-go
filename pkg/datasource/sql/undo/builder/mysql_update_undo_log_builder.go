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
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/util/bytes"
	"github.com/seata/seata-go/pkg/util/log"
)

const (
	maxInSize = 1000
)

func init() {
	undo.RegisterUndoLogBuilder(types.UpdateExecutor, GetMySQLUpdateUndoLogBuilder)
}

type MySQLUpdateUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetMySQLUpdateUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLUpdateUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
	}
}

func (u *MySQLUpdateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	if execCtx == nil || execCtx.ParseContext == nil || execCtx.ParseContext.UpdateStmt == nil {
		return nil, nil
	}

	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, 0)
		for _, param := range execCtx.NamedValues {
			vals = append(vals, param.Value)
		}
	}
	// use
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(ctx, execCtx, vals)
	if err != nil {
		return nil, err
	}

	tableName, _ := execCtx.ParseContext.GteTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, execCtx.DBName, tableName)
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

	image, err := u.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey2(image, *metaData)
	execCtx.TxCtx.LockKeys[lockKey] = struct{}{}
	image.SQLType = execCtx.ParseContext.SQLType

	return []*types.RecordImage{image}, nil
}

func (u *MySQLUpdateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if execCtx.ParseContext.UpdateStmt == nil {
		return nil, nil
	}

	var beforeImage *types.RecordImage
	if len(beforeImages) > 0 {
		beforeImage = beforeImages[0]
	}

	tableName, _ := execCtx.ParseContext.GteTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, execCtx.DBName, tableName)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs := u.buildAfterImageSQL(beforeImage, metaData)

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

	image, err := u.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey(rows, *metaData)
	execCtx.TxCtx.LockKeys[lockKey] = struct{}{}
	image.SQLType = execCtx.ParseContext.SQLType

	return []*types.RecordImage{image}, nil
}

func (u *MySQLUpdateUndoLogBuilder) buildAfterImageSQL(beforeImage *types.RecordImage, meta *types.TableMeta) (string, []driver.Value) {
	sb := strings.Builder{}
	// todo use ONLY_CARE_UPDATE_COLUMNS to judge select all columns or not
	sb.WriteString("SELECT * FROM " + meta.TableName + " WHERE ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), "mysql", maxInSize)
	sb.WriteString(" " + whereSQL + " ")
	return sb.String(), u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName())
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *MySQLUpdateUndoLogBuilder) buildBeforeImageSQL(ctx context.Context, execCtx *types.ExecContext, args []driver.Value) (string, []driver.Value, error) {
	updateStmt := execCtx.ParseContext.UpdateStmt
	if updateStmt == nil {
		log.Errorf("invalid update stmt")
		return "", nil, fmt.Errorf("invalid update stmt")
	}

	fields := make([]*ast.SelectField, 0, len(updateStmt.List))

	// todo use ONLY_CARE_UPDATE_COLUMNS to judge select all columns or not
	for _, column := range updateStmt.List {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: column.Column,
			},
		})
	}

	// select indexes columns
	tableName, _ := execCtx.ParseContext.GteTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, execCtx.DBName, tableName)
	if err != nil {
		return "", nil, err
	}
	for _, columnName := range metaData.GetPrimaryKeyOnlyName() {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.CIStr{
						O: columnName,
						L: columnName,
					},
				},
			},
		})
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           updateStmt.TableRefs,
		Where:          updateStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		OrderBy:        updateStmt.Order,
		Limit:          updateStmt.Limit,
		TableHints:     updateStmt.TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := bytes.NewByteBuffer([]byte{})
	_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	return sql, u.buildSelectArgs(&selStmt, args), nil
}

func (u *MySQLUpdateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.UpdateExecutor
}
