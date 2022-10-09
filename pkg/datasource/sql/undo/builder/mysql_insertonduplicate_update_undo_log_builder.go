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
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/util/log"
)

type MySQLInsertOnDuplicateUndoLogBuilder struct {
	BasicUndoLogBuilder
	BeforeSelectSql string
	Args            []driver.Value
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *exec.ExecContext) (*types.RecordImage, error) {
	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, len(execCtx.NamedValues))
		for n, param := range execCtx.NamedValues {
			vals[n] = param.Value
		}
	}
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(execCtx, vals)
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

	return u.buildRecordImages(rows, execCtx.MetaData)
}

// buildBeforeImageSQL build select sql from insert on duplicate update sql
func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildBeforeImageSQL(execCtx *exec.ExecContext, args []driver.Value) (string, []driver.Value, error) {
	p, err := parser.DoParser(execCtx.Query)
	if err != nil {
		return "", nil, err
	}
	if p.InsertStmt == nil {
		log.Errorf("invalid insert stmt")
		return "", nil, fmt.Errorf("invalid insert stmt")
	}
	if err = duplicateKeyUpdateCheck(p.InsertStmt, execCtx); err != nil {
		return "", nil, err
	}
	selectArgs := make([]driver.Value, 0)
	insertNum, paramMap := buildImageParameters(p.InsertStmt, args)

	sql := strings.Builder{}
	sql.WriteString("SELECT * FROM " + execCtx.MetaData.Name + " ")
	isContainWhere := false
	for i := 0; i < insertNum; i++ {
		finalI := i
		paramAppenderTempList := make([]driver.Value, 0)
		for _, index := range execCtx.MetaData.Indexs {
			//unique index
			if index.IType == types.IndexTypeUniqueKey {
				columnIsNull := true
				uniqueList := make([]string, 0)
				for _, columnMeta := range index.Values {
					columnName := columnMeta.Info.Name()
					imageParameters, ok := paramMap[columnName]
					if !ok && columnMeta.ColumnDef != "" {
						uniqueList = append(uniqueList, columnName+" = DEFAULT("+columnName+") ")
						columnIsNull = false
						continue
					}
					if (!ok && columnMeta.ColumnDef == "") || imageParameters[finalI] == nil {
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
		}
		selectArgs = append(selectArgs, paramAppenderTempList...)
	}
	log.Infof("build select sql by insert on duplicate sourceQuery, sql {}", sql.String())
	return sql.String(), selectArgs, nil
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) AfterImage(ctx context.Context, beforeImage types.RecordImage, execCtx *exec.ExecContext) (*types.RecordImage, error) {
	selectSQL, selectArgs := u.BeforeSelectSql, u.Args

	primaryValueMap := make(map[string][]interface{}, 0)
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
	stmt, err := execCtx.Conn.Prepare(afterImageSql.String())
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}

	return u.buildRecordImages(rows, execCtx.MetaData)
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) GetSQLType() types.SQLType {
	return types.SQLTypeInsert
}

func duplicateKeyUpdateCheck(insert *ast.InsertStmt, execCtx *exec.ExecContext) error {
	duplicateColsMap := make(map[string]bool, 0)
	for _, v := range insert.OnDuplicate {
		duplicateColsMap[v.Column.Name.O] = true
	}
	if len(duplicateColsMap) != 0 {
		for _, index := range execCtx.MetaData.Indexs {
			if "PRIMARY" == index.Name {
				for _, col := range index.Values {
					if duplicateColsMap[col.Info.Name()] {
						log.Errorf("update pk value is not supported!")
						return fmt.Errorf("update pk value is not supported! ")
					}
				}
			}
		}
	}
	return nil
}

// build sql params
func buildImageParameters(insert *ast.InsertStmt, args []driver.Value) (int, map[string][]driver.Value) {
	parameterMap := make(map[string][]driver.Value, 0)
	length := len(insert.OnDuplicate)
	args = args[:len(args)-length]

	rows := len(args) / len(insert.Columns)
	for i, column := range insert.Columns {
		for j := 0; j < rows; j++ {
			parameterMap[column.Name.O] = append(parameterMap[column.Name.O], args[i*j+i])
		}
	}
	return rows, parameterMap
}
