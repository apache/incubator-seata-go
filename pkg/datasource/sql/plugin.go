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

package sql

import (
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at"
	"seata.apache.org/seata-go/pkg/datasource/sql/hook"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/builder"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/mysql"
)

func Init() {
	hookRegister()
	executorRegister()
	undoInit()
	initDriver()
}

func hookRegister() {
	exec.RegisterHook(hook.NewLoggerSQLHook())
	exec.RegisterHook(hook.NewUndoLogSQLHook())
}

func executorRegister() {
	at.Init()
}

func undoInit() {
	mysqlUndoLogInit()
}

func mysqlUndoLogInit() {
	mysql.InitUndoLogManager()

	undo.RegisterUndoLogBuilder(types.DeleteExecutor, builder.GetMySQLDeleteUndoLogBuilder)
	undo.RegisterUndoLogBuilder(types.InsertExecutor, builder.GetMySQLInsertUndoLogBuilder)
	undo.RegisterUndoLogBuilder(types.InsertOnDuplicateExecutor, builder.GetMySQLInsertOnDuplicateUndoLogBuilder)
	undo.RegisterUndoLogBuilder(types.MultiExecutor, builder.GetMySQLMultiUndoLogBuilder)
	undo.RegisterUndoLogBuilder(types.UpdateExecutor, builder.GetMySQLUpdateUndoLogBuilder)
}
