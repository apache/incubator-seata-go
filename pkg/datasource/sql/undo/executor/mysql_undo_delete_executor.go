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

type mySQLUndoDeleteExecutor struct {
	baseExecutor *BaseExecutor
	sqlUndoLog   undo.SQLUndoLog
}

// newMySQLUndoDeleteExecutor init
func newMySQLUndoDeleteExecutor(sqlUndoLog undo.SQLUndoLog) *mySQLUndoDeleteExecutor {
	return &mySQLUndoDeleteExecutor{
		sqlUndoLog:   sqlUndoLog,
		baseExecutor: &BaseExecutor{sqlUndoLog: sqlUndoLog, undoImage: sqlUndoLog.AfterImage},
	}
}

func (m *mySQLUndoDeleteExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {

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
			if col.KeyType != types.PrimaryKey.Number() {
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

func (m *mySQLUndoDeleteExecutor) buildUndoSQL(dbType types.DBType) (string, error) {
	beforeImage := m.sqlUndoLog.BeforeImage
	rows := beforeImage.Rows
	if len(rows) == 0 {
		return "", fmt.Errorf("invalid undo log")
	}

	row := rows[0]
	fields := row.NonPrimaryKeys(row.Columns)
	pkList, err := GetOrderedPkList(beforeImage, row, dbType)
	if err != nil {
		return "", err
	}

	fields = append(fields, pkList...)

	var (
		insertColumns, insertValues         string
		insertColumnSlice, insertValueSlice []string
	)

	for key, _ := range fields {
		insertColumnSlice = append(insertColumnSlice, AddEscape(fields[key].ColumnName, dbType))
		insertValueSlice = append(insertValueSlice, "?")
	}

	insertColumns = strings.Join(insertColumnSlice, ", ")
	insertValues = strings.Join(insertValueSlice, ", ")

	// InsertSqlTemplate INSERT INTO a (x, y, z, pk) VALUES (?, ?, ?, ?)
	insertSqlTemplate := "INSERT INTO %s (%s) VALUES (%s)"
	return fmt.Sprintf(insertSqlTemplate, m.sqlUndoLog.TableName, insertColumns, insertValues), nil
}
