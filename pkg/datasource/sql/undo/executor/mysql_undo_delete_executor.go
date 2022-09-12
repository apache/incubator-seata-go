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

	"github.com/pkg/errors"
	sqlUtil "github.com/seata/seata-go/pkg/common/sql"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/impl"
)

type MySQLUndoDeleteExecutor struct {
	BaseExecutor *BaseExecutor
}

// NewMySQLUndoUpdateExecutor init
func NewMySQLUndoDeleteExecutor() *MySQLUndoUpdateExecutor {
	return &MySQLUndoUpdateExecutor{}
}

func (m *MySQLUndoDeleteExecutor) ExecuteOn(
	ctx context.Context,
	dbType types.DBType,
	sqlUndoLog impl.SQLUndoLog,
	conn *sql.Conn) error {

	m.BaseExecutor.ExecuteOn(ctx, dbType, sqlUndoLog, conn)
	undoSql, _ := m.buildUndoSQL(dbType, sqlUndoLog)

	stmt, err := conn.PrepareContext(ctx, undoSql)
	if err != nil {
		return err
	}

	beforeImage := sqlUndoLog.BeforeImage

	for _, row := range beforeImage.Rows {
		undoValues := make([]interface{}, 0)
		pkList, err := m.BaseExecutor.GetOrderedPkList(beforeImage, row, dbType)
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

func (m *MySQLUndoDeleteExecutor) buildUndoSQL(dbType types.DBType, sqlUndoLog impl.SQLUndoLog) (string, error) {
	beforeImage := sqlUndoLog.BeforeImage
	rows := beforeImage.Rows
	if len(rows) == 0 {
		return "", errors.New("invalid undo log")
	}

	row := rows[0]
	fields := row.NonPrimaryKeys(row.Columns)
	pkList, err := m.BaseExecutor.GetOrderedPkList(beforeImage, row, dbType)
	if err != nil {
		return "", err
	}

	fields = append(fields, pkList...)

	var insertColumns, insertValues string
	insertColumnSlice := make([]string, 0)
	insertValueSlice := make([]string, 0)

	for key, _ := range fields {
		insertColumnSlice = append(insertColumnSlice, sqlUtil.AddEscape(fields[key].Name, dbType))
		insertValueSlice = append(insertValueSlice, "?")
	}

	insertColumns = strings.Join(insertColumnSlice, ", ")
	insertValues = strings.Join(insertValueSlice, ", ")

	return fmt.Sprintf(constant.InsertSqlTemplate, sqlUndoLog.TableName, insertColumns, insertValues), nil
}
