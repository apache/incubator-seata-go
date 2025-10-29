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

type postgreSQLUndoUpdateExecutor struct {
	BaseExecutor *BaseExecutor
	sqlUndoLog   undo.SQLUndoLog
}

// newPostgreSQLUndoUpdateExecutor init
func newPostgreSQLUndoUpdateExecutor(sqlUndoLog undo.SQLUndoLog) *postgreSQLUndoUpdateExecutor {
	executor := &postgreSQLUndoUpdateExecutor{
		sqlUndoLog: sqlUndoLog,
	}
	executor.BaseExecutor = &BaseExecutor{
		sqlUndoLog: sqlUndoLog,
		undoImage:  sqlUndoLog.BeforeImage,
	}
	return executor
}

// ExecuteOn execute update undo logic
func (p *postgreSQLUndoUpdateExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
	log.Infof("PostgreSQL ExecuteOn called for UPDATE undo, table: %s", p.sqlUndoLog.TableName)

	// Validate data before undo
	valid, err := p.BaseExecutor.dataValidationAndGoOn(ctx, conn, dbType)
	if err != nil {
		log.Errorf("PostgreSQL undo update data validation error: %v", err)
		return err
	}
	if !valid {
		log.Warnf("PostgreSQL undo update skipped - data validation returned false for table %s", p.sqlUndoLog.TableName)
		return nil
	}

	undoSQL, undoValues := p.buildUndoSQL(dbType)

	log.Infof("PostgreSQL undo update SQL: %s, values: %v", undoSQL, undoValues)

	stmt, err := conn.PrepareContext(ctx, undoSQL)
	if err != nil {
		log.Errorf("PostgreSQL undo update prepare error: %v", err)
		return err
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, undoValues...)
	if err != nil {
		log.Errorf("PostgreSQL undo update exec error: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Infof("PostgreSQL undo update completed, rows affected: %d", rowsAffected)

	return nil
}

// buildUndoSQL build update SQL to restore before image
func (p *postgreSQLUndoUpdateExecutor) buildUndoSQL(dbType types.DBType) (string, []interface{}) {
	beforeImage := p.sqlUndoLog.BeforeImage
	afterImage := p.sqlUndoLog.AfterImage

	if beforeImage == nil || len(beforeImage.Rows) == 0 {
		return "", nil
	}

	// Use the original table name from TableMeta (preserves case from database)
	// PostgreSQL table names are case-insensitive by default, so we should use the original name
	tableName := beforeImage.TableMeta.TableName
	pkNameList := beforeImage.TableMeta.GetPrimaryKeyOnlyName()

	var updateSQL strings.Builder
	var params []interface{}

	updateSQL.WriteString("UPDATE ")
	// For PostgreSQL, use lowercase table name without quotes (standard convention)
	// This allows PostgreSQL to match tables created without quotes
	updateSQL.WriteString(strings.ToLower(tableName))
	updateSQL.WriteString(" SET ")

	setFields := make([]string, 0)
	paramIndex := 1

	for _, row := range beforeImage.Rows {
		for _, column := range row.Columns {
			isPK := false
			for _, pk := range pkNameList {
				if strings.EqualFold(column.ColumnName, pk) {
					isPK = true
					break
				}
			}
			if !isPK {
				// Use lowercase column names without quotes
				setFields = append(setFields, fmt.Sprintf(`%s = $%d`, strings.ToLower(column.ColumnName), paramIndex))
				params = append(params, column.Value)
				paramIndex++
			}
		}
		break
	}

	updateSQL.WriteString(strings.Join(setFields, ", "))
	updateSQL.WriteString(" WHERE ")

	whereConditions := make([]string, 0)
	for _, pk := range pkNameList {
		// Use lowercase primary key names without quotes
		whereConditions = append(whereConditions, fmt.Sprintf(`%s = $%d`, strings.ToLower(pk), paramIndex))
		paramIndex++
	}
	updateSQL.WriteString(strings.Join(whereConditions, " AND "))

	if afterImage != nil && len(afterImage.Rows) > 0 {
		for _, row := range afterImage.Rows {
			for _, pk := range pkNameList {
				for _, column := range row.Columns {
					if strings.EqualFold(column.ColumnName, pk) {
						params = append(params, column.Value)
						break
					}
				}
			}
			break
		}
	}

	return updateSQL.String(), params
}
