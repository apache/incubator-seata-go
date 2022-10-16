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

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
)

type MySQLUndoUpdateExecutor struct {
	BaseExecutor *BaseExecutor
}

// NewMySQLUndoUpdateExecutor init
func NewMySQLUndoUpdateExecutor() *MySQLUndoUpdateExecutor {
	return &MySQLUndoUpdateExecutor{}
}

func (m *MySQLUndoUpdateExecutor) ExecuteOn(
	ctx context.Context, dbType types.DBType,
	sqlUndoLog undo.SQLUndoLog, conn *sql.Conn) error {

	//m.BaseExecutor.ExecuteOn(ctx, dbType, sqlUndoLog, conn)
	undoSql, _ := m.buildUndoSQL(dbType, sqlUndoLog)

	stmt, err := conn.PrepareContext(ctx, undoSql)
	if err != nil {
		return err
	}

	beforeImage := sqlUndoLog.BeforeImage

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

		if _, err = stmt.ExecContext(ctx, undoValues); err != nil {
			return err
		}
	}

	return nil
}

// BuildUndoSQL
func (m *MySQLUndoUpdateExecutor) buildUndoSQL(dbType types.DBType, sqlUndoLog undo.SQLUndoLog) (string, error) {
	beforeImage := sqlUndoLog.BeforeImage
	rows := beforeImage.Rows
	row := rows[0]

	var (
		updateColumns                 string
		updateColumnSlice, pkNameList []string
	)

	nonPkFields := row.NonPrimaryKeys(row.Columns)
	for key, _ := range nonPkFields {
		updateColumnSlice = append(updateColumnSlice, AddEscape(nonPkFields[key].Name, dbType)+" = ? ")
	}

	updateColumns = strings.Join(updateColumnSlice, ", ")
	pkList, err := GetOrderedPkList(beforeImage, row, dbType)
	if err != nil {
		return "", err
	}

	for key, _ := range pkList {
		pkNameList = append(pkNameList, pkList[key].Name)
	}

	whereSql := BuildWhereConditionByPKs(pkNameList, dbType)

	// UpdateSqlTemplate UPDATE a SET x = ?, y = ?, z = ? WHERE pk1 in (?) pk2 in (?)
	updateSqlTemplate := "UPDATE %s SET %s WHERE %s "
	return fmt.Sprintf(updateSqlTemplate, sqlUndoLog.TableName, updateColumns, whereSql), nil
}
