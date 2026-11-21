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

package base

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestBatchDeleteUndoLog_PreservesCrossPairsAndIsIdempotent(t *testing.T) {
	tests := []struct {
		dbType types.DBType
		driver string
		dsnEnv string
	}{
		{types.DBTypeMySQL, "mysql", "SEATA_GO_TEST_MYSQL_DSN"},
		{types.DBTypePostgreSQL, "pgx", "SEATA_GO_TEST_POSTGRES_DSN"},
	}
	for _, tt := range tests {
		t.Run(tt.dbType.String(), func(t *testing.T) {
			dsn := os.Getenv(tt.dsnEnv)
			if dsn == "" {
				t.Skipf("%s is not set", tt.dsnEnv)
			}
			db, err := sql.Open(tt.driver, dsn)
			require.NoError(t, err)
			defer db.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			conn, err := db.Conn(ctx)
			require.NoError(t, err)
			defer conn.Close()

			// The fixture is visible only to this connection.
			const table = "undo_log_batch_delete_test"
			originalTable := undo.UndoConfig.LogTable
			undo.UndoConfig.LogTable = table
			t.Cleanup(func() { undo.UndoConfig.LogTable = originalTable })
			_, err = conn.ExecContext(ctx, "CREATE TEMPORARY TABLE "+table+
				" (xid VARCHAR(128) NOT NULL, branch_id BIGINT NOT NULL, UNIQUE (xid, branch_id))")
			require.NoError(t, err)
			_, err = conn.ExecContext(ctx, "INSERT INTO "+table+
				" (xid, branch_id) VALUES ('xid-a',1),('xid-b',2),('xid-a',2),('xid-b',1),('pending',3)")
			require.NoError(t, err)

			type undoKey struct {
				xid      string
				branchID int64
			}
			readRemaining := func() []undoKey {
				t.Helper()
				rows, err := conn.QueryContext(ctx, "SELECT xid, branch_id FROM "+table+" ORDER BY xid, branch_id")
				require.NoError(t, err)
				defer rows.Close()
				var remaining []undoKey
				for rows.Next() {
					var key undoKey
					require.NoError(t, rows.Scan(&key.xid, &key.branchID))
					remaining = append(remaining, key)
				}
				require.NoError(t, rows.Err())
				return remaining
			}
			want := []undoKey{{"pending", 3}, {"xid-a", 2}, {"xid-b", 1}}
			manager := NewBaseUndoLogManager(tt.dbType)
			for attempt := 1; attempt <= 2; attempt++ {
				require.NoError(t, manager.BatchDeleteUndoLog([]string{"xid-a", "xid-b"}, []int64{1, 2}, conn))
				require.Equal(t, want, readRemaining(), "cleanup attempt %d", attempt)
			}
		})
	}
}
