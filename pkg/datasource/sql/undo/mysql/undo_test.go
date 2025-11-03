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

package mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/base"
)


func TestNewUndoLogManager(t *testing.T) {
	manager := NewUndoLogManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.Base)
	assert.IsType(t, &base.BaseUndoLogManager{}, manager.Base)
}

func TestUndoLogManager_Init(t *testing.T) {
	manager := NewUndoLogManager()
	assert.NotPanics(t, func() {
		manager.Init()
	})
}

func TestUndoLogManager_DBType(t *testing.T) {
	manager := NewUndoLogManager()
	dbType := manager.DBType()
	assert.Equal(t, types.DBTypeMySQL, dbType)
}

func TestUndoLogManager_DeleteUndoLog(t *testing.T) {
	manager := NewUndoLogManager()
	
	assert.NotPanics(t, func() {
		manager.DeleteUndoLog(context.Background(), "test-xid", 123, nil)
	})
}

func TestUndoLogManager_BatchDeleteUndoLog(t *testing.T) {
	manager := NewUndoLogManager()
	
	assert.NotPanics(t, func() {
		manager.BatchDeleteUndoLog([]string{"xid1"}, []int64{123}, nil)
	})
}

func TestUndoLogManager_FlushUndoLog(t *testing.T) {
	manager := NewUndoLogManager()
	
	tranCtx := &types.TransactionContext{
		XID:      "test-xid",
		BranchID: 123,
	}
	
	assert.NotPanics(t, func() {
		manager.FlushUndoLog(tranCtx, nil)
	})
}

func TestUndoLogManager_RunUndo(t *testing.T) {
	manager := NewUndoLogManager()
	
	assert.NotPanics(t, func() {
		manager.RunUndo(context.Background(), "test-xid", 123, nil, "test_db")
	})
}

func TestUndoLogManager_HasUndoLogTable(t *testing.T) {
	manager := NewUndoLogManager()
	
	assert.NotPanics(t, func() {
		manager.HasUndoLogTable(context.Background(), nil)
	})
}

func TestUndoLogManager_ImplementsInterface(t *testing.T) {
	manager := NewUndoLogManager()
	assert.Implements(t, (*undo.UndoLogManager)(nil), manager)
}