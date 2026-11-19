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
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestAsyncWorker_Lifecycle(t *testing.T) {
	cfg := AsyncWorkerConfig{
		BufferLimit:            10,
		BufferCleanInterval:    time.Second,
		ReceiveChanSize:        10,
		CommitWorkerCount:      1,
		CommitWorkerBufferSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Pass nil as sourceManager since we don't process tasks in this test
	_ = NewAsyncWorker(ctx, prometheus.NewRegistry(), cfg, nil)

	// Allow it to run for a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop worker
	cancel()

	// Give it some time to exit (though we can't verify it easily without exposing internal state)
	time.Sleep(50 * time.Millisecond)
}

type deleteUndoManager struct {
	undo.UndoLogManager
	query string
}

func (m *deleteUndoManager) BatchDeleteUndoLog(xids []string, branchIDs []int64, conn *sql.Conn) error {
	_, err := conn.ExecContext(context.Background(), m.query, xids[0], branchIDs[0])
	return err
}

func TestAsyncWorkerUsesResourceUndoManagerAndRetriesDeletion(t *testing.T) {
	manager := &ATSourceManager{}
	worker := &AsyncWorker{
		resourceMgr:      manager,
		commitQueue:      make(chan phaseTwoContext, 2),
		rePutBackToQueue: prometheus.NewCounter(prometheus.CounterOpts{Name: "undo_retry_test"}),
	}
	addResource := func(id, query string) sqlmock.Sqlmock {
		t.Helper()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		manager.resourceCache.Store(id, &DBResource{
			db: db, dbType: types.DBTypeMySQL, undoLogMgr: &deleteUndoManager{query: query},
		})
		return mock
	}
	mockA := addResource("a", "DELETE FROM undo_a")
	mockB := addResource("b", "DELETE FROM undo_b")
	a := phaseTwoContext{ResourceID: "a", Xid: "xid-a", BranchID: 1}
	b := phaseTwoContext{ResourceID: "b", Xid: "xid-b", BranchID: 2}
	mockA.ExpectExec("DELETE FROM undo_a").WithArgs(a.Xid, a.BranchID).WillReturnError(errors.New("delete failed"))
	mockB.ExpectExec("DELETE FROM undo_b").WithArgs(b.Xid, b.BranchID).WillReturnResult(sqlmock.NewResult(0, 1))
	mockA.ExpectExec("DELETE FROM undo_a").WithArgs(a.Xid, a.BranchID).WillReturnResult(sqlmock.NewResult(0, 1))

	worker.dealWithGroupedContexts(a.ResourceID, []phaseTwoContext{a})
	worker.dealWithGroupedContexts(b.ResourceID, []phaseTwoContext{b})
	require.Len(t, worker.commitQueue, 1)
	retry := <-worker.commitQueue
	require.Equal(t, a, retry)
	worker.dealWithGroupedContexts(retry.ResourceID, []phaseTwoContext{retry})
	require.Empty(t, worker.commitQueue)
	require.NoError(t, mockA.ExpectationsWereMet())
	require.NoError(t, mockB.ExpectationsWereMet())
}
