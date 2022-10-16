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
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
)

type MySQLUndoDeleteExecutor struct {
	BaseExecutor *BaseExecutor
}

// NewMySQLUndoDeleteExecutor init
func NewMySQLUndoDeleteExecutor() *MySQLUndoUpdateExecutor {
	return &MySQLUndoUpdateExecutor{}
}

func (m *MySQLUndoDeleteExecutor) ExecuteOn(
	ctx context.Context, dbType types.DBType,
	sqlUndoLog undo.SQLUndoLog, conn *sql.Conn) error {

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
			if col.KeyType != types.PrimaryKey.String() {
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

func (m *MySQLUndoDeleteExecutor) buildUndoSQL(dbType types.DBType, sqlUndoLog undo.SQLUndoLog) (string, error) {
	beforeImage := sqlUndoLog.BeforeImage
	rows := beforeImage.Rows
	if len(rows) == 0 {
		return "", errors.New("invalid undo log")
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
		insertColumnSlice = append(insertColumnSlice, AddEscape(fields[key].Name, dbType))
		insertValueSlice = append(insertValueSlice, "?")
	}

	insertColumns = strings.Join(insertColumnSlice, ", ")
	insertValues = strings.Join(insertValueSlice, ", ")

	// InsertSqlTemplate INSERT INTO a (x, y, z, pk) VALUES (?, ?, ?, ?)
	insertSqlTemplate := "INSERT INTO %s (%s) VALUES (%s)"
	return fmt.Sprintf(insertSqlTemplate, sqlUndoLog.TableName, insertColumns, insertValues), nil
}
