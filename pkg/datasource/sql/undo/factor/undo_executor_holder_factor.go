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
	"errors"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/executor"
)

var undoExecutorHolderMap map[types.DBType]undo.UndoExecutorHolder

var ErrNotImplDBType = errors.New("db type executor not implement")

// GetUndoExecutorHolder get exactly executor holder
func GetUndoExecutorHolder(dbType types.DBType) (undo.UndoExecutorHolder, error) {
	// lazy init
	if undoExecutorHolderMap == nil {
		undoExecutorHolderMap = map[types.DBType]undo.UndoExecutorHolder{
			// todo impl oracle, mariadb, postgresql etc ...
			types.DBTypeMySQL: executor.NewMySQLUndoExecutorHolder(),
		}
	}

	if executorHolder, ok := undoExecutorHolderMap[dbType]; ok {
		return executorHolder, nil
	} else {
		return nil, ErrNotImplDBType
	}
}
