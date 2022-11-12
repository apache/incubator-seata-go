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
)

import (
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/test_driver"
)

import (
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/util/log"
)

func init() {
	undo.RegisterUndoLogBuilder(types.InsertOnDuplicateExecutor, GetMySQLInsertOnDuplicateUndoLogBuilder)
}

type MySQLInsertOnDuplicateUndoLogBuilder struct {
	BasicUndoLogBuilder
	BeforeSelectSql string
	Args            []driver.Value
}

func GetMySQLInsertOnDuplicateUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLInsertOnDuplicateUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
		BeforeSelectSql:     "",
		Args:                make([]driver.Value, 0),
	}
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.InsertOnDuplicateExecutor
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	if execCtx.ParseContext.InsertStmt == nil {
		log.Errorf("invalid insert stmt")
		return nil, fmt.Errorf("invalid insert stmt")
	}
	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, len(execCtx.NamedValues))
		for n, param := range execCtx.NamedValues {
			vals[n] = param.Value
		}
	}
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(execCtx.ParseContext.InsertStmt, metaData, vals)
	if err != nil {
		return nil, err
	}
	u.BeforeSelectSql = selectSQL
	u.Args = selectArgs
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
	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}
	return []*types.RecordImage{image}, nil
}

// buildBeforeImageSQL build select sql from insert on duplicate update sql
func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildBeforeImageSQL(insertStmt *ast.InsertStmt, metaData types.TableMeta, args []driver.Value) (string, []driver.Value, error) {
	if err := checkDuplicateKeyUpdate(insertStmt, metaData); err != nil {
		return "", nil, err
	}
	var selectArgs []driver.Value
	insertNum, paramMap := u.buildImageParameters(insertStmt, args)

	sql := strings.Builder{}
	sql.WriteString("SELECT * FROM " + metaData.Name + " ")
	isContainWhere := false
	for i := 0; i < insertNum; i++ {
		finalI := i
		paramAppenderTempList := make([]driver.Value, 0)
		for _, index := range metaData.Indexs {
			//unique index
			if index.IType != types.IndexUnique && index.IType != types.IndexTypePrimaryKey {
				continue
			}
			columnIsNull := true
			uniqueList := make([]string, 0)
			for _, columnMeta := range index.Values {
				columnName := columnMeta.ColumnName
				imageParameters, ok := paramMap[columnName]
				if !ok && columnMeta.ColumnDef != nil {
					uniqueList = append(uniqueList, columnName+" = DEFAULT("+columnName+") ")
					columnIsNull = false
					continue
				}
				if (!ok && columnMeta.ColumnDef == nil) || imageParameters[finalI] == nil {
					if !strings.EqualFold("PRIMARY", index.Name) {
						columnIsNull = false
						uniqueList = append(uniqueList, columnName+" is ? ")
						paramAppenderTempList = append(paramAppenderTempList, "NULL")
						continue
					}
					break
				}
				columnIsNull = false
				uniqueList = append(uniqueList, columnName+" = ? ")
				paramAppenderTempList = append(paramAppenderTempList, imageParameters[finalI])
			}

			if !columnIsNull {
				if isContainWhere {
					sql.WriteString(" OR (" + strings.Join(uniqueList, " and ") + ") ")
				} else {
					sql.WriteString(" WHERE (" + strings.Join(uniqueList, " and ") + ") ")
					isContainWhere = true
				}
			}
		}
		selectArgs = append(selectArgs, paramAppenderTempList...)
	}
	log.Infof("build select sql by insert on duplicate sourceQuery, sql {}", sql.String())
	return sql.String(), selectArgs, nil
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	afterSelectSql, selectArgs := u.buildAfterImageSQL(ctx, beforeImages)
	stmt, err := execCtx.Conn.Prepare(afterSelectSql)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}
	return []*types.RecordImage{image}, nil
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildAfterImageSQL(ctx context.Context, beforeImages []*types.RecordImage) (string, []driver.Value) {
	selectSQL, selectArgs := u.BeforeSelectSql, u.Args

	var beforeImage *types.RecordImage
	if len(beforeImages) > 0 {
		beforeImage = beforeImages[0]
	}
	primaryValueMap := make(map[string][]interface{})
	for _, row := range beforeImage.Rows {
		for _, col := range row.Columns {
			if col.KeyType == types.IndexTypePrimaryKey {
				primaryValueMap[col.Name] = append(primaryValueMap[col.Name], col.Value)
			}
		}
	}

	var afterImageSql strings.Builder
	var primaryValues []driver.Value
	afterImageSql.WriteString(selectSQL)
	for i := 0; i < len(beforeImage.Rows); i++ {
		wherePrimaryList := make([]string, 0)
		for name, value := range primaryValueMap {
			wherePrimaryList = append(wherePrimaryList, name+" = ? ")
			primaryValues = append(primaryValues, value[i])
		}
		afterImageSql.WriteString(" OR (" + strings.Join(wherePrimaryList, " and ") + ") ")
	}
	if len(primaryValues) == 0 {
		afterImageSql.WriteString(" FOR UPDATE")
	}
	selectArgs = append(selectArgs, primaryValues...)
	log.Infof("build after select sql by insert on duplicate sourceQuery, sql {}", afterImageSql.String())
	return afterImageSql.String(), selectArgs
}

func checkDuplicateKeyUpdate(insert *ast.InsertStmt, metaData types.TableMeta) error {
	duplicateColsMap := make(map[string]bool)
	for _, v := range insert.OnDuplicate {
		duplicateColsMap[v.Column.Name.L] = true
	}
	if len(duplicateColsMap) == 0 {
		return nil
	}
	for _, index := range metaData.Indexs {
		if types.IndexTypePrimaryKey != index.IType {
			continue
		}
		for name, col := range index.Values {
			if duplicateColsMap[strings.ToLower(col.ColumnName)] {
				log.Errorf("update pk value is not supported! index name:%s update column name: %s", name, col.ColumnName)
				return fmt.Errorf("update pk value is not supported! ")
			}
		}
	}
	return nil
}

// build sql params
func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildImageParameters(insert *ast.InsertStmt, args []driver.Value) (int, map[string][]driver.Value) {
	var (
		parameterMap = make(map[string][]driver.Value)
		selectArgs   = make([]driver.Value, 0)
	)
	// values (?,?,?),(?,?,"Jack")
	for _, list := range insert.Lists {
		for _, node := range list {
			buildSelectArgs(node, args, &selectArgs)
		}
	}
	rows := len(insert.Lists)
	for i, column := range insert.Columns {
		for j := 0; j < rows; j++ {
			parameterMap[column.Name.L] = append(parameterMap[column.Name.L], selectArgs[i+j*len(insert.Columns)])
		}
	}
	return rows, parameterMap
}

func buildSelectArgs(node ast.Node, args []driver.Value, selectArgs *[]driver.Value) {
	if node == nil {
		return
	}
	switch node.(type) {
	case *test_driver.ValueExpr:
		*selectArgs = append(*selectArgs, node.(*test_driver.ValueExpr).Datum.GetValue())
		break
	case *test_driver.ParamMarkerExpr:
		*selectArgs = append(*selectArgs, args[int32(node.(*test_driver.ParamMarkerExpr).Order)])
		break
	}
}
