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

package factor

import (
	"fmt"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/log"
)

func GetUndoExecutor(dbType types.DBType, sqlUndoLog undo.SQLUndoLog) (res undo.UndoExecutor, err error) {
	undoExecutorHolder, err := GetUndoExecutorHolder(dbType)
	if err != nil {
		log.Errorf("[GetUndoExecutor] get undo executor holder fail, err: %v", err)
		return nil, err
	}

	switch sqlUndoLog.SQLType {
	case types.SQLTypeInsert:
		res = undoExecutorHolder.GetInsertExecutor(sqlUndoLog)
	case types.SQLTypeDelete:
		res = undoExecutorHolder.GetDeleteExecutor(sqlUndoLog)
	case types.SQLTypeUpdate:
		res = undoExecutorHolder.GetUpdateExecutor(sqlUndoLog)
	default:
		return nil, fmt.Errorf("sql type: %d not support", sqlUndoLog.SQLType)
	}

	return
}
