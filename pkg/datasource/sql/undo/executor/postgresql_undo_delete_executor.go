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

package executor

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/log"
)

type postgreSQLUndoDeleteExecutor struct {
	BaseExecutor *BaseExecutor
	sqlUndoLog   undo.SQLUndoLog
}

// newPostgreSQLUndoDeleteExecutor init
func newPostgreSQLUndoDeleteExecutor(sqlUndoLog undo.SQLUndoLog) *postgreSQLUndoDeleteExecutor {
	executor := &postgreSQLUndoDeleteExecutor{
		sqlUndoLog: sqlUndoLog,
	}
	executor.BaseExecutor = &BaseExecutor{
		sqlUndoLog: sqlUndoLog,
		undoImage:  sqlUndoLog.BeforeImage,
	}
	return executor
}

// ExecuteOn execute delete undo logic
func (p *postgreSQLUndoDeleteExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
	valid, err := p.BaseExecutor.dataValidationAndGoOn(ctx, conn, dbType)
	if err != nil {
		return err
	}
	if !valid {
		return nil
	}

	undoSQL, undoValues := p.buildUndoSQL(dbType)

	log.Infof("PostgreSQL undo delete SQL: %s, values: %v", undoSQL, undoValues)

	stmt, err := conn.PrepareContext(ctx, undoSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, undoValues...)
	if err != nil {
		return err
	}

	return nil
}

// buildUndoSQL build insert SQL to restore deleted data
func (p *postgreSQLUndoDeleteExecutor) buildUndoSQL(dbType types.DBType) (string, []interface{}) {
	beforeImage := p.sqlUndoLog.BeforeImage
	if beforeImage == nil || len(beforeImage.Rows) == 0 {
		return "", nil
	}

	tableName := beforeImage.TableName
	var insertSQL strings.Builder
	var params []interface{}

	insertSQL.WriteString("INSERT INTO ")
	insertSQL.WriteString(fmt.Sprintf(`"%s"`, tableName))
	insertSQL.WriteString(" (")

	var columns []string
	paramIndex := 1

	if len(beforeImage.Rows) > 0 {
		for _, column := range beforeImage.Rows[0].Columns {
			columns = append(columns, fmt.Sprintf(`"%s"`, column.ColumnName))
		}
	}

	insertSQL.WriteString(strings.Join(columns, ", "))
	insertSQL.WriteString(") VALUES ")

	var valuesClauses []string
	for _, row := range beforeImage.Rows {
		var rowPlaceholders []string
		for _, column := range row.Columns {
			rowPlaceholders = append(rowPlaceholders, fmt.Sprintf("$%d", paramIndex))
			params = append(params, column.Value)
			paramIndex++
		}
		valuesClauses = append(valuesClauses, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	insertSQL.WriteString(strings.Join(valuesClauses, ", "))

	pkNameList := beforeImage.TableMeta.GetPrimaryKeyOnlyName()
	if len(pkNameList) > 0 {
		insertSQL.WriteString(" ON CONFLICT (")
		var quotedPKs []string
		for _, pk := range pkNameList {
			quotedPKs = append(quotedPKs, fmt.Sprintf(`"%s"`, pk))
		}
		insertSQL.WriteString(strings.Join(quotedPKs, ", "))
		insertSQL.WriteString(") DO UPDATE SET ")

		var setFields []string
		for _, column := range beforeImage.Rows[0].Columns {
			isPK := false
			for _, pk := range pkNameList {
				if strings.EqualFold(column.ColumnName, pk) {
					isPK = true
					break
				}
			}
			if !isPK {
				setFields = append(setFields, fmt.Sprintf(`"%s" = EXCLUDED."%s"`, column.ColumnName, column.ColumnName))
			}
		}
		if len(setFields) > 0 {
			insertSQL.WriteString(strings.Join(setFields, ", "))
		} else {
			insertSQL.WriteString(fmt.Sprintf(`"%s" = EXCLUDED."%s"`, pkNameList[0], pkNameList[0]))
		}
	}

	return insertSQL.String(), params
}
