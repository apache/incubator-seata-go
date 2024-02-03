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

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

type mySQLUndoInsertExecutor struct {
	BaseExecutor *BaseExecutor
	sqlUndoLog   undo.SQLUndoLog
}

// newMySQLUndoInsertExecutor init
func newMySQLUndoInsertExecutor(sqlUndoLog undo.SQLUndoLog) *mySQLUndoInsertExecutor {
	return &mySQLUndoInsertExecutor{sqlUndoLog: sqlUndoLog}
}

// ExecuteOn execute insert undo logic
func (m *mySQLUndoInsertExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {

	if err := m.BaseExecutor.ExecuteOn(ctx, dbType, conn); err != nil {
		return err
	}

	// build delete sql
	undoSql, _ := m.buildUndoSQL(dbType)

	stmt, err := conn.PrepareContext(ctx, undoSql)
	if err != nil {
		return err
	}

	afterImage := m.sqlUndoLog.AfterImage
	for _, row := range afterImage.Rows {
		pkValueList := make([]interface{}, 0)

		for _, col := range row.Columns {
			if col.KeyType == types.PrimaryKey.Number() {
				pkValueList = append(pkValueList, col.Value)
			}
		}

		if _, err = stmt.Exec(pkValueList...); err != nil {
			return err
		}
	}

	return nil
}

// buildUndoSQL build insert undo log
func (m *mySQLUndoInsertExecutor) buildUndoSQL(dbType types.DBType) (string, error) {
	afterImage := m.sqlUndoLog.AfterImage
	rows := afterImage.Rows
	if len(rows) == 0 {
		return "", fmt.Errorf("invalid undo log")
	}

	str, err := m.generateDeleteSql(afterImage, rows, dbType, m.sqlUndoLog)
	if err != nil {
		return "", err
	}

	return str, nil
}

// generateDeleteSql generate delete sql
func (m *mySQLUndoInsertExecutor) generateDeleteSql(
	image *types.RecordImage, rows []types.RowImage,
	dbType types.DBType, sqlUndoLog undo.SQLUndoLog) (string, error) {

	colImages, err := GetOrderedPkList(image, rows[0], dbType)
	if err != nil {
		return "", err
	}

	var pkList []string
	for key, _ := range colImages {
		pkList = append(pkList, colImages[key].ColumnName)
	}

	whereSql := BuildWhereConditionByPKs(pkList, dbType)

	deleteSqlTemplate := "DELETE FROM %s WHERE %s "
	return fmt.Sprintf(deleteSqlTemplate, sqlUndoLog.TableName, whereSql), nil
}
