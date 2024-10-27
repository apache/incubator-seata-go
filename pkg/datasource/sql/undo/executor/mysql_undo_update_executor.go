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
)

type mySQLUndoUpdateExecutor struct {
	baseExecutor *BaseExecutor
	sqlUndoLog   undo.SQLUndoLog
}

// newMySQLUndoUpdateExecutor init
func newMySQLUndoUpdateExecutor(sqlUndoLog undo.SQLUndoLog) *mySQLUndoUpdateExecutor {
	return &mySQLUndoUpdateExecutor{
		sqlUndoLog:   sqlUndoLog,
		baseExecutor: &BaseExecutor{sqlUndoLog: sqlUndoLog, undoImage: sqlUndoLog.AfterImage},
	}
}

func (m *mySQLUndoUpdateExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
	ok, err := m.baseExecutor.dataValidationAndGoOn(ctx, conn)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	undoSql, _ := m.buildUndoSQL(dbType)
	stmt, err := conn.PrepareContext(ctx, undoSql)
	if err != nil {
		return err
	}

	beforeImage := m.sqlUndoLog.BeforeImage
	for _, row := range beforeImage.Rows {
		undoValues := make([]interface{}, 0)
		pkList, err := GetOrderedPkList(beforeImage, row, dbType)
		if err != nil {
			return err
		}

		for _, col := range row.Columns {
			if col.KeyType != types.IndexTypePrimaryKey {
				undoValues = append(undoValues, col.Value)
			}
		}

		for _, col := range pkList {
			undoValues = append(undoValues, col.Value)
		}

		if _, err = stmt.Exec(undoValues...); err != nil {
			return err
		}
	}

	return nil
}

// BuildUndoSQL
func (m *mySQLUndoUpdateExecutor) buildUndoSQL(dbType types.DBType) (string, error) {
	beforeImage := m.sqlUndoLog.BeforeImage
	rows := beforeImage.Rows
	row := rows[0]

	var (
		updateColumns                 string
		updateColumnSlice, pkNameList []string
	)

	nonPkFields := row.NonPrimaryKeys(row.Columns)
	for key, _ := range nonPkFields {
		updateColumnSlice = append(updateColumnSlice, AddEscape(nonPkFields[key].ColumnName, dbType)+" = ? ")
	}

	updateColumns = strings.Join(updateColumnSlice, ", ")
	pkList, err := GetOrderedPkList(beforeImage, row, dbType)
	if err != nil {
		return "", err
	}

	for key, _ := range pkList {
		pkNameList = append(pkNameList, pkList[key].ColumnName)
	}

	whereSql := BuildWhereConditionByPKs(pkNameList, dbType)

	// UpdateSqlTemplate UPDATE a SET x = ?, y = ?, z = ? WHERE pk1 in (?) pk2 in (?)
	updateSqlTemplate := "UPDATE %s SET %s WHERE %s "
	return fmt.Sprintf(updateSqlTemplate, m.sqlUndoLog.TableName, updateColumns, whereSql), nil
}
