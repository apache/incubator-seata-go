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
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

type PostgreSQLUndoExecutorHolder struct{}

func NewPostgreSQLUndoExecutorHolder() undo.UndoExecutorHolder {
	return &PostgreSQLUndoExecutorHolder{}
}

func (p *PostgreSQLUndoExecutorHolder) GetInsertExecutor(sqlUndoLog undo.SQLUndoLog) undo.UndoExecutor {
	executor := newMySQLUndoInsertExecutor(sqlUndoLog)
	executor.BaseExecutor.dbType = types.DBTypePostgreSQL
	return executor
}

func (p *PostgreSQLUndoExecutorHolder) GetUpdateExecutor(sqlUndoLog undo.SQLUndoLog) undo.UndoExecutor {
	executor := newMySQLUndoUpdateExecutor(sqlUndoLog)
	executor.baseExecutor.dbType = types.DBTypePostgreSQL
	return executor
}

func (p *PostgreSQLUndoExecutorHolder) GetDeleteExecutor(sqlUndoLog undo.SQLUndoLog) undo.UndoExecutor {
	executor := newMySQLUndoDeleteExecutor(sqlUndoLog)
	executor.baseExecutor.dbType = types.DBTypePostgreSQL
	return executor
}
