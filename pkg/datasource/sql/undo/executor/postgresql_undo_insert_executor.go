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

type postgreSQLUndoInsertExecutor struct {
	BaseExecutor *BaseExecutor
	sqlUndoLog   undo.SQLUndoLog
}

// newPostgreSQLUndoInsertExecutor init
func newPostgreSQLUndoInsertExecutor(sqlUndoLog undo.SQLUndoLog) *postgreSQLUndoInsertExecutor {
	executor := &postgreSQLUndoInsertExecutor{
		sqlUndoLog: sqlUndoLog,
	}
	executor.BaseExecutor = &BaseExecutor{
		sqlUndoLog: sqlUndoLog,
		undoImage:  sqlUndoLog.AfterImage,
	}
	return executor
}

// ExecuteOn execute insert undo logic
func (p *postgreSQLUndoInsertExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
	// Validate data before undo
	valid, err := p.BaseExecutor.dataValidationAndGoOn(ctx, conn, dbType)
	if err != nil {
		return err
	}
	if !valid {
		return nil
	}

	undoSQL, undoValues := p.buildUndoSQL(dbType)

	log.Infof("PostgreSQL undo insert SQL: %s, values: %v", undoSQL, undoValues)

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

// buildUndoSQL build delete SQL for undo insert operation
func (p *postgreSQLUndoInsertExecutor) buildUndoSQL(dbType types.DBType) (string, []interface{}) {
	afterImage := p.sqlUndoLog.AfterImage
	if afterImage == nil || len(afterImage.Rows) == 0 {
		return "", nil
	}

	tableName := afterImage.TableName
	pkNameList := afterImage.TableMeta.GetPrimaryKeyOnlyName()

	var deleteSQL strings.Builder
	deleteSQL.WriteString("DELETE FROM ")
	deleteSQL.WriteString(fmt.Sprintf(`"%s"`, tableName))
	deleteSQL.WriteString(" WHERE ")

	where := buildWhereConditionByPKs(pkNameList, len(afterImage.Rows), maxInSize, dbType)
	deleteSQL.WriteString(where)

	params := buildPKParams(afterImage.Rows, pkNameList)

	sql := deleteSQL.String()
	sql = convertToPostgreSQLParams(sql, len(params))

	return sql, params
}

// convertToPostgreSQLParams converts ? placeholders to PostgreSQL $1, $2, ... format
func convertToPostgreSQLParams(sql string, paramCount int) string {
	result := sql
	paramIndex := 1
	
	for strings.Contains(result, "?") && paramIndex <= paramCount {
		result = strings.Replace(result, "?", fmt.Sprintf("$%d", paramIndex), 1)
		paramIndex++
	}
	
	return result
}
