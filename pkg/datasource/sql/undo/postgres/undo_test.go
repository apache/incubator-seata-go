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

package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/base"
)

func TestNewUndoLogManager(t *testing.T) {
	manager := NewUndoLogManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.Base)
	assert.IsType(t, &base.BaseUndoLogManager{}, manager.Base)
}

func TestUndoLogManager_DBType(t *testing.T) {
	manager := NewUndoLogManager()
	assert.Equal(t, types.DBTypePostgreSQL, manager.DBType())
	assert.Equal(t, types.DBTypePostgreSQL, manager.Base.DBType())
}

func TestUndoLogManager_DeleteUndoLog_RewritesPlaceholders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()
	ctx := context.Background()

	mock.ExpectPrepare(`DELETE FROM (.+) WHERE branch_id = \$1 AND xid = \$2`).
		ExpectExec().
		WithArgs(int64(123), "test-xid").
		WillReturnResult(sqlmock.NewResult(0, 1))

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	err = manager.DeleteUndoLog(ctx, "test-xid", 123, conn)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_ImplementsInterface(t *testing.T) {
	manager := NewUndoLogManager()
	assert.Implements(t, (*undo.UndoLogManager)(nil), manager)
}
