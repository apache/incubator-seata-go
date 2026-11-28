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
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func addCleanupTestResource(t *testing.T, manager *ATSourceManager, resourceID string, matcher sqlmock.QueryMatcher) sqlmock.Sqlmock {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	manager.resourceCache.Store(resourceID, &DBResource{dbType: types.DBTypeMySQL, db: db})
	return mock
}

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

func TestAsyncWorker_RetryDoesNotBlockNextTask(t *testing.T) {
	manager := &ATSourceManager{}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var firstAttempt sync.Once
	failing := addCleanupTestResource(t, manager, "failing", sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		firstAttempt.Do(func() {
			close(firstStarted)
			<-releaseFirst
		})
		return sqlmock.QueryMatcherRegexp.Match(expected, actual)
	}))
	healthy := addCleanupTestResource(t, manager, "healthy", sqlmock.QueryMatcherRegexp)
	for _, result := range []error{errors.New("lock wait timeout"), nil} {
		exec := failing.ExpectPrepare("DELETE FROM undo_log").ExpectExec().
			WithArgs("xid-fail", int64(1), "xid-fail", int64(2))
		if result != nil {
			exec.WillReturnError(result)
		} else {
			exec.WillReturnResult(sqlmock.NewResult(0, 2))
		}
	}
	for branchID := int64(1); branchID <= 4; branchID += 2 {
		healthy.ExpectPrepare("DELETE FROM undo_log").ExpectExec().
			WithArgs("xid-ok", branchID, "xid-ok", branchID+1).
			WillReturnResult(sqlmock.NewResult(0, 2))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer func() {
		select {
		case <-releaseFirst:
		default:
			close(releaseFirst)
		}
	}()
	worker := NewAsyncWorker(ctx, prometheus.NewRegistry(), AsyncWorkerConfig{
		BufferLimit:            3,
		BufferCleanInterval:    250 * time.Millisecond,
		ReceiveChanSize:        1,
		CommitWorkerCount:      1,
		CommitWorkerBufferSize: 1,
	}, manager)
	queue := func(resourceID, xid string, count int64) {
		t.Helper()
		for branchID := int64(1); branchID <= count; branchID++ {
			status, err := worker.BranchCommit(ctx, rm.BranchResource{Xid: xid, BranchId: branchID, ResourceId: resourceID})
			require.NoError(t, err)
			require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), status)
		}
	}
	queue("failing", "xid-fail", 2)
	select {
	case <-firstStarted:
	case <-ctx.Done():
		t.Fatal("first cleanup never started")
	}
	// Fill the Fanout buffer and leave a further batch waiting for a slot.
	queue("healthy", "xid-ok", 4)
	close(releaseFirst)

	require.Eventually(t, func() bool {
		return healthy.ExpectationsWereMet() == nil && failing.ExpectationsWereMet() == nil
	}, 3*time.Second, 10*time.Millisecond, "queued cleanup and failed batch retry must both complete")
	require.Equal(t, float64(2), testutil.ToFloat64(worker.rePutBackToQueue))
}

func TestAsyncWorker_UnavailableResourceDoesNotBlockHealthyResource(t *testing.T) {
	manager := &ATSourceManager{}
	healthy := addCleanupTestResource(t, manager, "healthy", sqlmock.QueryMatcherRegexp)
	healthy.ExpectPrepare("DELETE FROM undo_log").ExpectExec().
		WithArgs("xid-healthy", int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	worker := NewAsyncWorker(ctx, prometheus.NewRegistry(), AsyncWorkerConfig{
		BufferLimit:            3,
		BufferCleanInterval:    50 * time.Millisecond,
		ReceiveChanSize:        1,
		CommitWorkerCount:      1,
		CommitWorkerBufferSize: 1,
	}, manager)
	manager.worker = worker
	unavailable := rm.BranchResource{Xid: "xid-unavailable", ResourceId: "unavailable"}
	// Three batches of two plus the entry queue's one slot.
	const unavailableBranches = 7
	for id := int64(1); id <= unavailableBranches; id++ {
		unavailable.BranchId = id
		status, err := manager.BranchCommit(ctx, unavailable)
		require.NoError(t, err)
		require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), status)
	}
	require.Eventually(t, func() bool {
		return testutil.ToFloat64(worker.rePutBackToQueue) >= unavailableBranches
	}, time.Second, 5*time.Millisecond)

	unavailable.BranchId++
	status, err := manager.BranchCommit(ctx, unavailable)
	require.ErrorContains(t, err, "async undo log cleanup capacity exhausted")
	require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitFailedRetryable), status)

	status, err = manager.BranchCommit(ctx, rm.BranchResource{Xid: "xid-healthy", BranchId: 1, ResourceId: "healthy"})
	require.NoError(t, err)
	require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), status)
	require.Eventually(t, func() bool {
		return healthy.ExpectationsWereMet() == nil
	}, time.Second, 5*time.Millisecond, "healthy cleanup did not start")
	require.Eventually(t, func() bool {
		worker.pendingMu.Lock()
		defer worker.pendingMu.Unlock()
		return worker.pendingBranches == unavailableBranches &&
			len(worker.pendingByResource) == 1 && worker.pendingByResource["unavailable"] == unavailableBranches
	}, time.Second, 5*time.Millisecond, "successful cleanup must release its slot and preserve failed branches")
}

func TestAsyncWorker_SlowResourceDoesNotBlockOtherResourceCommit(t *testing.T) {
	manager := &ATSourceManager{}
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	var slowOnce sync.Once
	slow := addCleanupTestResource(t, manager, "slow", sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		slowOnce.Do(func() {
			close(slowStarted)
			<-releaseSlow
		})
		return errors.New("slow resource unavailable")
	}))
	slow.ExpectPrepare("DELETE FROM undo_log")
	healthy := addCleanupTestResource(t, manager, "healthy", sqlmock.QueryMatcherRegexp)
	healthy.ExpectPrepare("DELETE FROM undo_log").ExpectExec().
		WithArgs("xid-healthy", int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseSlow) }) }
	defer release()
	worker := NewAsyncWorker(ctx, prometheus.NewRegistry(), AsyncWorkerConfig{
		BufferLimit:            3,
		BufferCleanInterval:    50 * time.Millisecond,
		ReceiveChanSize:        1,
		CommitWorkerCount:      1,
		CommitWorkerBufferSize: 1,
	}, manager)

	// Fill the slow resource's whole share: one running job, one buffered job, a full batch and the entry queue.
	slowBranches := int64(worker.perResourceLimit)
	for branchID := int64(1); branchID <= slowBranches; branchID++ {
		status, err := worker.BranchCommit(ctx, rm.BranchResource{Xid: "xid-slow", BranchId: branchID, ResourceId: "slow"})
		require.NoError(t, err)
		require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), status)
		if branchID == 1 {
			select {
			case <-slowStarted:
			case <-ctx.Done():
				t.Fatal("slow cleanup never started")
			}
		}
	}

	commitCtx, commitCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer commitCancel()
	status, err := worker.BranchCommit(commitCtx, rm.BranchResource{Xid: "xid-healthy", BranchId: 1, ResourceId: "healthy"})
	require.NoError(t, err, "a slow resource must not block admission of another resource")
	require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), status)

	release()
	require.Eventually(t, func() bool {
		return healthy.ExpectationsWereMet() == nil
	}, 3*time.Second, 10*time.Millisecond, "healthy cleanup must run once workers free up")
}

func TestAsyncWorker_EmptyResourceIDIsDropped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker := NewAsyncWorker(ctx, prometheus.NewRegistry(), AsyncWorkerConfig{
		BufferLimit:            3,
		BufferCleanInterval:    time.Hour,
		ReceiveChanSize:        1,
		CommitWorkerCount:      1,
		CommitWorkerBufferSize: 1,
	}, &ATSourceManager{})

	// More than the entry queue holds, so a leaked slot or queued entry would block or exhaust capacity.
	for branchID := int64(1); branchID <= 3; branchID++ {
		status, err := worker.BranchCommit(ctx, rm.BranchResource{Xid: "xid-without-resource", BranchId: branchID})
		require.NoError(t, err)
		require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), status)
	}

	require.Equal(t, float64(3), testutil.ToFloat64(worker.droppedNoResourceID))
	require.Equal(t, float64(0), testutil.ToFloat64(worker.rePutBackToQueue))
	require.Empty(t, worker.commitQueue)
	worker.pendingMu.Lock()
	defer worker.pendingMu.Unlock()
	require.Zero(t, worker.pendingBranches)
	require.Empty(t, worker.pendingByResource)
}

func TestAsyncWorker_BatchUndoLogCleanup(t *testing.T) {
	manager := &ATSourceManager{}
	primary := addCleanupTestResource(t, manager, "primary", sqlmock.QueryMatcherRegexp)
	secondary := addCleanupTestResource(t, manager, "secondary", sqlmock.QueryMatcherRegexp)

	makeBranch := func(resourceID string, branchID int64) phaseTwoContext {
		return phaseTwoContext{
			Xid:        fmt.Sprintf("xid-%d", branchID),
			BranchID:   branchID,
			ResourceID: resourceID,
		}
	}
	const fullBatchSize = 100
	const tailSize = 50
	primaryBranches := make([]phaseTwoContext, 0, 2*fullBatchSize+tailSize)
	for branchID := int64(1); branchID <= 2*fullBatchSize+tailSize; branchID++ {
		primaryBranches = append(primaryBranches, makeBranch("primary", branchID))
	}
	expectDelete := func(mock sqlmock.Sqlmock, batch []phaseTwoContext) *sqlmock.ExpectedExec {
		t.Helper()
		args := make([]driver.Value, 0, len(batch)*2)
		for _, branch := range batch {
			args = append(args, branch.Xid, branch.BranchID)
		}
		return mock.ExpectPrepare("DELETE FROM undo_log").ExpectExec().WithArgs(args...)
	}
	wantRetries := primaryBranches[fullBatchSize : 2*fullBatchSize]
	secondaryBranches := []phaseTwoContext{makeBranch("secondary", 1)}
	expectDelete(primary, primaryBranches[:fullBatchSize]).WillReturnResult(sqlmock.NewResult(0, fullBatchSize))
	expectDelete(primary, wantRetries).WillReturnError(errors.New("temporary delete failure"))
	expectDelete(primary, primaryBranches[2*fullBatchSize:]).WillReturnResult(sqlmock.NewResult(0, tailSize))
	expectDelete(secondary, secondaryBranches).WillReturnResult(sqlmock.NewResult(0, 1))

	worker := &AsyncWorker{resourceMgr: manager}
	result := worker.cleanUndoLogs(context.Background(), append(secondaryBranches, primaryBranches...))

	require.NoError(t, primary.ExpectationsWereMet())
	require.NoError(t, secondary.ExpectationsWereMet())
	require.Equal(t, wantRetries, result.failed)
	require.Equal(t, map[string]int{"primary": fullBatchSize + tailSize, "secondary": 1}, result.cleanedByResource)
}
