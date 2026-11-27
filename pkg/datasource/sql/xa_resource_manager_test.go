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
	gosql "database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/bluele/gcache"
	"github.com/go-sql-driver/mysql"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
	gettyrm "seata.apache.org/seata-go/v2/pkg/rm/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

type phaseTwoTestDriverConn struct{}

func (c *phaseTwoTestDriverConn) Prepare(query string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (c *phaseTwoTestDriverConn) Close() error {
	return nil
}

func (c *phaseTwoTestDriverConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not supported")
}

type phaseTwoTestXAResource struct {
	commitErr      error
	rollbackErr    error
	classification xa.PhaseTwoErrorClassification
	classifiedAs   xa.PhaseTwoOperation
	commitCalls    int
	endCalls       int
	prepareCalls   int
	rollbackCalls  int
}

func (r *phaseTwoTestXAResource) Commit(ctx context.Context, xid string, onePhase bool) error {
	r.commitCalls++
	return r.commitErr
}

func (r *phaseTwoTestXAResource) End(ctx context.Context, xid string, flags int) error {
	r.endCalls++
	return nil
}

func (r *phaseTwoTestXAResource) Forget(ctx context.Context, xid string) error {
	return nil
}

func (r *phaseTwoTestXAResource) GetTransactionTimeout() time.Duration {
	return 0
}

func (r *phaseTwoTestXAResource) IsSameRM(ctx context.Context, resource xa.XAResource) bool {
	return false
}

func (r *phaseTwoTestXAResource) XAPrepare(ctx context.Context, xid string) error {
	r.prepareCalls++
	return nil
}

func (r *phaseTwoTestXAResource) Recover(ctx context.Context, flag int) ([]string, error) {
	return nil, nil
}

func (r *phaseTwoTestXAResource) Rollback(ctx context.Context, xid string) error {
	r.rollbackCalls++
	return r.rollbackErr
}

func (r *phaseTwoTestXAResource) SetTransactionTimeout(duration time.Duration) bool {
	return false
}

func (r *phaseTwoTestXAResource) Start(ctx context.Context, xid string, flags int) error {
	return nil
}

func (r *phaseTwoTestXAResource) IsAlreadyEnded(err error) bool {
	return false
}

func (r *phaseTwoTestXAResource) ClassifyPhaseTwoError(
	operation xa.PhaseTwoOperation,
	err error,
) xa.PhaseTwoErrorClassification {
	r.classifiedAs = operation
	return r.classification
}

// blockingPhaseTwoTestXAResource lets a test hold the first phase-two call
// while a duplicate callback reaches the same keeper entry. This makes the
// second callback exercise the lock-protected terminal-cache check instead of
// merely taking the fast path before it opens a connection.
type blockingPhaseTwoTestXAResource struct {
	phaseTwoTestXAResource
	commitEntered   chan struct{}
	releaseCommit   chan struct{}
	commitOnce      sync.Once
	rollbackEntered chan struct{}
	releaseRollback chan struct{}
	rollbackOnce    sync.Once
}

func (r *blockingPhaseTwoTestXAResource) Commit(ctx context.Context, xid string, onePhase bool) error {
	r.commitCalls++
	r.commitOnce.Do(func() { close(r.commitEntered) })
	<-r.releaseCommit
	return r.commitErr
}

func (r *blockingPhaseTwoTestXAResource) Rollback(ctx context.Context, xid string) error {
	r.rollbackCalls++
	if r.rollbackEntered == nil {
		return r.rollbackErr
	}
	r.rollbackOnce.Do(func() { close(r.rollbackEntered) })
	<-r.releaseRollback
	return r.rollbackErr
}

type classifiedPhaseTwoTestXAResource struct {
	phaseTwoTestXAResource
	operationErr  error
	calls         int32
	active        int32
	secondEntered chan struct{}
	releaseSecond chan struct{}
}

func (resource *classifiedPhaseTwoTestXAResource) Commit(context.Context, string, bool) error {
	return resource.execute()
}

func (resource *classifiedPhaseTwoTestXAResource) Rollback(context.Context, string) error {
	return resource.execute()
}

func (resource *classifiedPhaseTwoTestXAResource) execute() error {
	if atomic.AddInt32(&resource.calls, 1) == 1 {
		return resource.operationErr
	}
	atomic.StoreInt32(&resource.active, 1)
	close(resource.secondEntered)
	<-resource.releaseSecond
	atomic.StoreInt32(&resource.active, 0)
	return nil
}

type blockingPhaseTwoTestClassifier struct {
	phaseTwoTestXAResource
	entered chan struct{}
	release chan struct{}
}

func (classifier *blockingPhaseTwoTestClassifier) ClassifyPhaseTwoError(operation xa.PhaseTwoOperation, err error) xa.PhaseTwoErrorClassification {
	classifier.classifiedAs = operation
	close(classifier.entered)
	<-classifier.release
	return classifier.classification
}

type phaseTwoCloseTrackingConn struct {
	closeErrorDriverConn
	active         *int32
	closedDuringIO int32
}

func (connection *phaseTwoCloseTrackingConn) Close() error {
	if atomic.LoadInt32(connection.active) != 0 {
		atomic.StoreInt32(&connection.closedDuringIO, 1)
	}
	return connection.closeErrorDriverConn.Close()
}

type phaseTwoWaitContext struct {
	context.Context
	entered chan struct{}
	once    sync.Once
}

func (ctx *phaseTwoWaitContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.entered) })
	return ctx.Context.Done()
}

type phaseTwoCallbackResult struct {
	status branch.BranchStatus
	err    error
}

func awaitPhaseTwoResult(t *testing.T, results <-chan phaseTwoCallbackResult) phaseTwoCallbackResult {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("phase-two callback did not finish")
		return phaseTwoCallbackResult{}
	}
}

func awaitPhaseTwoWorkers(t *testing.T, workers *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("phase-two callbacks did not finish after releasing all barriers")
	}
}

func newPhaseTwoTestManager(
	t *testing.T,
	resource interface {
		xa.XAResource
		xa.XAErrorClassifier
	},
	classifiers ...xa.XAErrorClassifier,
) (*XAResourceManager, rm.BranchResource, string) {
	t.Helper()

	branchStatusCache = gcache.New(16).LRU().Expiration(time.Minute).Build()

	const (
		resourceID = "mysql://127.0.0.1/test"
		globalXID  = "127.0.0.1:8091:1001"
		branchID   = int64(2001)
	)

	dbResource := &DBResource{
		resourceID:   resourceID,
		dbType:       types.DBTypeMySQL,
		shouldBeHeld: true,
	}
	var classifier xa.XAErrorClassifier = resource
	if len(classifiers) > 0 {
		classifier = classifiers[0]
	}
	xaID := XaIdBuild(globalXID, uint64(branchID))
	xaConn := &XAConn{
		Conn: &Conn{
			targetConn: &phaseTwoTestDriverConn{},
			res:        dbResource,
			dbType:     types.DBTypeMySQL,
		},
		xaResource:        resource,
		xaErrorClassifier: classifier,
		xaBranchXid:       xaID,
		shouldBeHeld:      true,
		isConnKept:        true,
		prepareTime:       time.Now(),
		xaActive:          false,
		rollBacked:        false,
	}
	assert.NoError(t, dbResource.Hold(xaID.String(), xaConn))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(resourceID, dbResource)

	return manager, rm.BranchResource{
		BranchType: branch.BranchTypeXA,
		Xid:        globalXID,
		BranchId:   branchID,
		ResourceId: resourceID,
	}, xaID.String()
}

func assertPhaseTwoConnectionHeld(
	t *testing.T,
	manager *XAResourceManager,
	resource rm.BranchResource,
	xaID string,
	want bool,
) {
	t.Helper()

	value, ok := manager.resourceCache.Load(resource.ResourceId)
	if !assert.True(t, ok) {
		return
	}
	dbResource, ok := value.(*DBResource)
	if !assert.True(t, ok) {
		return
	}
	_, held := dbResource.Lookup(xaID)
	assert.Equal(t, want, held)
}

func TestXAResourceManager_LockQuery(t *testing.T) {
	tests := []struct {
		name    string
		resp    interface{}
		respErr error
		want    bool
		wantErr string
	}{
		{
			name: "lockable",
			resp: message.GlobalLockQueryResponse{Lockable: true},
			want: true,
		},
		{
			name: "unlockable",
			resp: message.GlobalLockQueryResponse{Lockable: false},
			want: false,
		},
		{
			name:    "remoting error",
			respErr: errors.New("network timeout"),
			want:    false,
			wantErr: "network timeout",
		},
	}

	param := rm.LockQueryParam{
		BranchType: branch.BranchTypeXA,
		ResourceId: "jdbc:mysql://test/resource",
		Xid:        "test-xid",
		LockKeys:   "user:1",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(getty.GetGettyRemotingClient()), "SendSyncRequest",
				func(_ *getty.GettyRemotingClient, msg interface{}) (interface{}, error) {
					req, ok := msg.(message.GlobalLockQueryRequest)
					if assert.True(t, ok) {
						assert.Equal(t, param.BranchType, req.BranchType)
						assert.Equal(t, param.ResourceId, req.ResourceId)
						assert.Equal(t, param.Xid, req.Xid)
						assert.Equal(t, param.LockKeys, req.LockKey)
					}
					return tt.resp, tt.respErr
				})
			defer patches.Reset()

			xaManager := &XAResourceManager{rmRemoting: &gettyrm.GettyRMRemoting{}}

			got, err := xaManager.LockQuery(context.Background(), param)

			assert.Equal(t, tt.want, got)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestXAResourceManager_CloseTimedOutPhaseTwoConnections(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name              string
		prepared          bool
		holdUntilPhaseTwo bool
		preparedAt        time.Time
		wantClosed        bool
	}{
		{
			name:       "unprepared keeper entry is ignored",
			preparedAt: time.Time{},
		},
		{
			name:              "mandatory owner is retained",
			prepared:          true,
			holdUntilPhaseTwo: true,
			preparedAt:        now.Add(-time.Minute),
		},
		{
			name:       "recent detachable branch is retained",
			prepared:   true,
			preparedAt: now.Add(-500 * time.Millisecond),
		},
		{
			name:       "expired detachable branch is force closed",
			prepared:   true,
			preparedAt: now.Add(-2 * time.Second),
			wantClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			underlying := &closeErrorDriverConn{}
			xaID := XaIdBuild("127.0.0.1:8091:1001", 123)
			resource := &DBResource{
				resourceID:   "mysql://127.0.0.1/test",
				dbType:       types.DBTypeMySQL,
				keepInKeeper: true,
			}
			xaConn := &XAConn{
				Conn: &Conn{
					targetConn: underlying,
					res:        resource,
					dbType:     types.DBTypeMySQL,
				},
				xaBranchXid:         xaID,
				keepInKeeper:        true,
				shouldBeHeld:        tt.holdUntilPhaseTwo,
				isConnKept:          true,
				preparedForPhaseTwo: tt.prepared,
				prepareTime:         tt.preparedAt,
			}
			assert.NoError(t, resource.Hold(xaID.String(), xaConn))

			manager := &XAResourceManager{
				config: XAConfig{TwoPhaseHoldTime: time.Second},
			}
			manager.resourceCache.Store(resource.resourceID, resource)

			manager.closeTimedOutPhaseTwoConnections(now)

			if tt.wantClosed {
				assert.Equal(t, int32(1), atomic.LoadInt32(&underlying.closeCalls))
				_, held := resource.Lookup(xaID.String())
				assert.False(t, held)
			} else {
				assert.Equal(t, int32(0), atomic.LoadInt32(&underlying.closeCalls))
				_, held := resource.Lookup(xaID.String())
				assert.True(t, held)
			}
		})
	}
}

func TestXAResourceManager_UnknownDatabaseRetainsPreparedOwnerAfterTimeout(t *testing.T) {
	now := time.Now()
	underlying := &closeErrorDriverConn{}
	resource := &DBResource{resourceID: "unknown-resource", dbType: types.DBTypeUnknown}
	xaID := XaIdBuild("unknown-database", 1)
	connection := &XAConn{
		Conn:                &Conn{targetConn: underlying, res: resource},
		xaBranchXid:         xaID,
		preparedForPhaseTwo: true,
		prepareTime:         now.Add(-time.Minute),
	}
	connection.configureConnectionHold(context.Background())
	connection.keepIfNecessary()
	require.True(t, connection.ShouldBeHeld())
	require.True(t, connection.isKept())
	manager := &XAResourceManager{config: XAConfig{TwoPhaseHoldTime: time.Second}}
	manager.resourceCache.Store(resource.resourceID, resource)

	manager.closeTimedOutPhaseTwoConnections(now)

	assert.Equal(t, int32(0), atomic.LoadInt32(&underlying.closeCalls))
	owner, held := resource.Lookup(xaID.String())
	assert.True(t, held)
	assert.Same(t, connection, owner)
	assert.True(t, connection.preparedForPhaseTwo)
}

func TestXAResourceManager_BranchCommit_StatusAndCache(t *testing.T) {
	t.Run("success caches committed terminal status", func(t *testing.T) {
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{})

		status, err := manager.BranchCommit(context.Background(), resource)

		assert.NoError(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})

	t.Run("failure stays retryable and does not cache success", func(t *testing.T) {
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{
			commitErr: errors.New("temporary commit failure"),
		})

		status, err := manager.BranchCommit(context.Background(), resource)

		assert.EqualError(t, err, "temporary commit failure")
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedRetryable, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusUnknown, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, true)
	})
}

func TestXAResourceManager_BranchRollback_StatusAndCache(t *testing.T) {
	t.Run("success caches rollbacked terminal status", func(t *testing.T) {
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{})

		status, err := manager.BranchRollback(context.Background(), resource)

		assert.NoError(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})

	t.Run("failure stays retryable and does not cache success", func(t *testing.T) {
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{
			rollbackErr: errors.New("temporary rollback failure"),
		})

		status, err := manager.BranchRollback(context.Background(), resource)

		assert.EqualError(t, err, "temporary rollback failure")
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbackFailedRetryable, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusUnknown, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, true)
	})
}

func TestXAResourceManager_BranchCommit_ClassifiesTerminalErrors(t *testing.T) {
	t.Run("already committed converges successfully", func(t *testing.T) {
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{
			commitErr:      errors.New("already committed"),
			classification: xa.PhaseTwoAlreadyCommitted,
		})

		status, err := manager.BranchCommit(context.Background(), resource)

		assert.NoError(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})

	t.Run("already rollbacked is an unretryable commit failure", func(t *testing.T) {
		commitErr := errors.New("branch already rollbacked")
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{
			commitErr:      commitErr,
			classification: xa.PhaseTwoAlreadyRolledBack,
		})

		status, err := manager.BranchCommit(context.Background(), resource)

		assert.ErrorIs(t, err, commitErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedUnretryable, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})

	t.Run("protocol error is unretryable without inventing a terminal outcome", func(t *testing.T) {
		commitErr := errors.New("invalid XA arguments")
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{
			commitErr:      commitErr,
			classification: xa.PhaseTwoUnretryable,
		})

		status, err := manager.BranchCommit(context.Background(), resource)

		assert.ErrorIs(t, err, commitErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedUnretryable, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusUnknown, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})
}

func TestXAResourceManager_BranchRollback_ClassifiesTerminalErrors(t *testing.T) {
	t.Run("already rollbacked converges successfully", func(t *testing.T) {
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{
			rollbackErr:    errors.New("already rollbacked"),
			classification: xa.PhaseTwoAlreadyRolledBack,
		})

		status, err := manager.BranchRollback(context.Background(), resource)

		assert.NoError(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})

	t.Run("already committed is an unretryable rollback failure", func(t *testing.T) {
		rollbackErr := errors.New("branch already committed")
		manager, resource, xaID := newPhaseTwoTestManager(t, &phaseTwoTestXAResource{
			rollbackErr:    rollbackErr,
			classification: xa.PhaseTwoAlreadyCommitted,
		})

		status, err := manager.BranchRollback(context.Background(), resource)

		assert.ErrorIs(t, err, rollbackErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbackFailedUnretryable, status)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})
}

func TestXAResourceManager_MySQLXAERNOTAIsDirectionAware(t *testing.T) {
	t.Run("rollback treats missing XID as idempotent success", func(t *testing.T) {
		nota := &mysql.MySQLError{
			Number:  types.ErrCodeXAER_NOTA,
			Message: "XAER_NOTA: Unknown XID",
		}
		xaResource := &phaseTwoTestXAResource{rollbackErr: nota}
		manager, resource, xaID := newPhaseTwoTestManager(
			t,
			xaResource,
			&xa.MysqlXAErrorClassifier{},
		)

		status, err := manager.BranchRollback(context.Background(), resource)

		assert.NoError(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
		assert.Equal(t, 1, xaResource.rollbackCalls)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, false)
	})

	t.Run("commit keeps missing XID ambiguous and retryable", func(t *testing.T) {
		nota := &mysql.MySQLError{
			Number:  types.ErrCodeXAER_NOTA,
			Message: "XAER_NOTA: Unknown XID",
		}
		xaResource := &phaseTwoTestXAResource{commitErr: nota}
		manager, resource, xaID := newPhaseTwoTestManager(
			t,
			xaResource,
			&xa.MysqlXAErrorClassifier{},
		)

		status, err := manager.BranchCommit(context.Background(), resource)

		assert.ErrorIs(t, err, nota)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedRetryable, status)
		assert.Equal(t, 1, xaResource.commitCalls)
		cached, cacheErr := branchStatus(xaID)
		assert.NoError(t, cacheErr)
		assert.EqualValues(t, branch.BranchStatusUnknown, cached)
		assertPhaseTwoConnectionHeld(t, manager, resource, xaID, true)
	})
}

func TestXAResourceManager_TimeoutRollbackThenTCRollbackTreatsNOTAAsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	branchStatusCache = gcache.New(16).LRU().Expiration(time.Minute).Build()
	previousTimeout := xaConnTimeout
	xaConnTimeout = time.Second
	defer func() {
		xaConnTimeout = previousTimeout
	}()

	const (
		resourceID = "mysql://127.0.0.1/timeout-test"
		globalXID  = "127.0.0.1:8091:3001"
		branchID   = int64(4001)
	)
	xaID := XaIdBuild(globalXID, uint64(branchID))

	phaseTwoConn := mock.NewMockTestDriverConn(ctrl)
	phaseTwoConn.EXPECT().
		ExecContext(
			gomock.Any(),
			"XA ROLLBACK '"+xaID.String()+"'",
			gomock.Any(),
		).
		Return(
			driver.ResultNoRows,
			&mysql.MySQLError{
				Number:  types.ErrCodeXAER_NOTA,
				Message: "XAER_NOTA: Unknown XID",
			},
		).
		Times(1)
	phaseTwoConn.EXPECT().Close().Return(nil).Times(1)

	connector := mock.NewMockTestDriverConnector(ctrl)
	connector.EXPECT().Connect(gomock.Any()).Return(phaseTwoConn, nil).Times(1)

	dbResource := &DBResource{
		resourceID:   resourceID,
		connector:    connector,
		dbType:       types.DBTypeMySQL,
		keepInKeeper: true,
		shouldBeHeld: true,
	}
	localResource := &phaseTwoTestXAResource{}
	owner := &XAConn{
		Conn: &Conn{
			targetConn: &phaseTwoTestDriverConn{},
			res:        dbResource,
			txCtx:      types.NewTxCtx(),
			autoCommit: false,
			dbType:     types.DBTypeMySQL,
		},
		xaResource:         localResource,
		xaErrorClassifier:  localResource,
		xaBranchXid:        xaID,
		xaActive:           true,
		branchRegisterTime: time.Now().Add(-2 * time.Second),
		keepInKeeper:       true,
		shouldBeHeld:       true,
		isConnKept:         true,
	}
	owner.txCtx.XID = globalXID
	owner.txCtx.BranchID = uint64(branchID)
	assert.NoError(t, dbResource.Hold(xaID.String(), owner))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(resourceID, dbResource)
	branchResource := rm.BranchResource{
		BranchType: branch.BranchTypeXA,
		Xid:        globalXID,
		BranchId:   branchID,
		ResourceId: resourceID,
	}

	timeoutErr := owner.Commit(context.Background())

	if assert.Error(t, timeoutErr) {
		assert.Contains(t, timeoutErr.Error(), "XA branch timeout error")
		assert.NotContains(t, timeoutErr.Error(), "XAER_NOTA")
	}
	assert.Equal(t, 1, localResource.rollbackCalls)
	assert.Equal(t, 0, localResource.prepareCalls)
	_, ownerStillHeld := dbResource.Lookup(xaID.String())
	assert.False(t, ownerStillHeld)

	status, err := manager.BranchRollback(context.Background(), branchResource)

	assert.NoError(t, err)
	assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
	cached, cacheErr := branchStatus(xaID.String())
	assert.NoError(t, cacheErr)
	assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, cached)
}

func TestXAResourceManager_CachedTerminalStatusMakesPhaseTwoIdempotent(t *testing.T) {
	t.Run("repeated commit does not hit the database", func(t *testing.T) {
		xaResource := &phaseTwoTestXAResource{}
		manager, resource, _ := newPhaseTwoTestManager(t, xaResource)

		firstStatus, firstErr := manager.BranchCommit(context.Background(), resource)
		assert.NoError(t, firstErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, firstStatus)

		xaResource.commitErr = errors.New("database should not be called again")
		secondStatus, secondErr := manager.BranchCommit(context.Background(), resource)
		assert.NoError(t, secondErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, secondStatus)
		assert.Equal(t, 1, xaResource.commitCalls)
	})

	t.Run("opposite retry returns an unretryable conflict", func(t *testing.T) {
		xaResource := &phaseTwoTestXAResource{}
		manager, resource, _ := newPhaseTwoTestManager(t, xaResource)

		commitStatus, commitErr := manager.BranchCommit(context.Background(), resource)
		assert.NoError(t, commitErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, commitStatus)

		rollbackStatus, rollbackErr := manager.BranchRollback(context.Background(), resource)
		assert.Error(t, rollbackErr)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbackFailedUnretryable, rollbackStatus)
		assert.Equal(t, 0, xaResource.rollbackCalls)
	})
}

func TestXAResourceManager_ConcurrentPhaseTwoCallbacksAreIdempotent(t *testing.T) {
	t.Run("same-direction commit executes once", func(t *testing.T) {
		xaResource := &blockingPhaseTwoTestXAResource{
			commitEntered: make(chan struct{}),
			releaseCommit: make(chan struct{}),
		}
		manager, resource, _ := newPhaseTwoTestManager(t, xaResource)

		type result struct {
			status branch.BranchStatus
			err    error
		}
		results := make(chan result, 2)
		go func() {
			status, err := manager.BranchCommit(context.Background(), resource)
			results <- result{status: status, err: err}
		}()
		<-xaResource.commitEntered
		go func() {
			status, err := manager.BranchCommit(context.Background(), resource)
			results <- result{status: status, err: err}
		}()

		close(xaResource.releaseCommit)
		first := <-results
		second := <-results
		assert.NoError(t, first.err)
		assert.NoError(t, second.err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, first.status)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, second.status)
		assert.Equal(t, 1, xaResource.commitCalls)
	})

	t.Run("same-direction rollback executes once", func(t *testing.T) {
		xaResource := &blockingPhaseTwoTestXAResource{
			rollbackEntered: make(chan struct{}),
			releaseRollback: make(chan struct{}),
		}
		manager, resource, _ := newPhaseTwoTestManager(t, xaResource)

		type result struct {
			status branch.BranchStatus
			err    error
		}
		results := make(chan result, 2)
		go func() {
			status, err := manager.BranchRollback(context.Background(), resource)
			results <- result{status: status, err: err}
		}()
		<-xaResource.rollbackEntered
		go func() {
			status, err := manager.BranchRollback(context.Background(), resource)
			results <- result{status: status, err: err}
		}()

		close(xaResource.releaseRollback)
		first := <-results
		second := <-results
		assert.NoError(t, first.err)
		assert.NoError(t, second.err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, first.status)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, second.status)
		assert.Equal(t, 1, xaResource.rollbackCalls)
	})

	t.Run("opposite-direction callback observes committed terminal state", func(t *testing.T) {
		xaResource := &blockingPhaseTwoTestXAResource{
			commitEntered: make(chan struct{}),
			releaseCommit: make(chan struct{}),
		}
		manager, resource, _ := newPhaseTwoTestManager(t, xaResource)

		type commitResult struct {
			status branch.BranchStatus
			err    error
		}
		commitDone := make(chan commitResult, 1)
		go func() {
			status, err := manager.BranchCommit(context.Background(), resource)
			commitDone <- commitResult{status: status, err: err}
		}()
		<-xaResource.commitEntered

		rollbackDone := make(chan commitResult, 1)
		go func() {
			status, err := manager.BranchRollback(context.Background(), resource)
			rollbackDone <- commitResult{status: status, err: err}
		}()
		close(xaResource.releaseCommit)

		commit := <-commitDone
		rollback := <-rollbackDone
		assert.NoError(t, commit.err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, commit.status)
		assert.Error(t, rollback.err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbackFailedUnretryable, rollback.status)
		assert.Equal(t, 1, xaResource.commitCalls)
		assert.Equal(t, 0, xaResource.rollbackCalls)
	})
}

func TestXAResourceManager_TerminalClassificationSerializesCallbacks(t *testing.T) {
	for _, test := range []struct {
		name           string
		firstOperation xa.PhaseTwoOperation
		nextOperation  xa.PhaseTwoOperation
		classification xa.PhaseTwoErrorClassification
		firstStatus    branch.BranchStatus
		nextStatus     branch.BranchStatus
		firstErr       bool
		nextErr        bool
	}{
		{
			name: "duplicate commit", firstOperation: xa.PhaseTwoCommit, nextOperation: xa.PhaseTwoCommit,
			classification: xa.PhaseTwoAlreadyCommitted,
			firstStatus:    branch.BranchStatusPhasetwoCommitted, nextStatus: branch.BranchStatusPhasetwoCommitted,
		},
		{
			name: "duplicate rollback", firstOperation: xa.PhaseTwoRollback, nextOperation: xa.PhaseTwoRollback,
			classification: xa.PhaseTwoAlreadyRolledBack,
			firstStatus:    branch.BranchStatusPhasetwoRollbacked, nextStatus: branch.BranchStatusPhasetwoRollbacked,
		},
		{
			name: "commit discovers rollback", firstOperation: xa.PhaseTwoCommit, nextOperation: xa.PhaseTwoRollback,
			classification: xa.PhaseTwoAlreadyRolledBack,
			firstStatus:    branch.BranchStatusPhasetwoCommitFailedUnretryable, nextStatus: branch.BranchStatusPhasetwoRollbacked,
			firstErr: true,
		},
		{
			name: "rollback discovers commit", firstOperation: xa.PhaseTwoRollback, nextOperation: xa.PhaseTwoCommit,
			classification: xa.PhaseTwoAlreadyCommitted,
			firstStatus:    branch.BranchStatusPhasetwoRollbackFailedUnretryable, nextStatus: branch.BranchStatusPhasetwoCommitted,
			firstErr: true,
		},
		{
			name: "rollback follows committed result", firstOperation: xa.PhaseTwoCommit, nextOperation: xa.PhaseTwoRollback,
			classification: xa.PhaseTwoAlreadyCommitted,
			firstStatus:    branch.BranchStatusPhasetwoCommitted, nextStatus: branch.BranchStatusPhasetwoRollbackFailedUnretryable,
			nextErr: true,
		},
		{
			name: "commit follows rollbacked result", firstOperation: xa.PhaseTwoRollback, nextOperation: xa.PhaseTwoCommit,
			classification: xa.PhaseTwoAlreadyRolledBack,
			firstStatus:    branch.BranchStatusPhasetwoRollbacked, nextStatus: branch.BranchStatusPhasetwoCommitFailedUnretryable,
			nextErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, discarded := range []bool{false, true} {
				name := "pool owns connection"
				if discarded {
					name = "pool discarded connection"
				}
				t.Run(name, func(t *testing.T) {
					operationErr := errors.New("terminal XA outcome")
					resource := &classifiedPhaseTwoTestXAResource{
						operationErr: operationErr, secondEntered: make(chan struct{}), releaseSecond: make(chan struct{}),
					}
					classifier := &blockingPhaseTwoTestClassifier{
						phaseTwoTestXAResource: phaseTwoTestXAResource{classification: test.classification},
						entered:                make(chan struct{}), release: make(chan struct{}),
					}
					manager, branchResource, xaID := newPhaseTwoTestManager(t, resource, classifier)
					cachedResource, found := manager.resourceCache.Load(branchResource.ResourceId)
					require.True(t, found)
					cachedConn, held := cachedResource.(*DBResource).Lookup(xaID)
					require.True(t, held)
					connection := cachedConn.(*XAConn)
					physical := &phaseTwoCloseTrackingConn{active: &resource.active}
					connection.targetConn = physical
					connection.poolDiscarded = discarded

					var workers sync.WaitGroup
					var classificationOnce, secondOnce sync.Once
					releaseClassification := func() { classificationOnce.Do(func() { close(classifier.release) }) }
					releaseSecond := func() { secondOnce.Do(func() { close(resource.releaseSecond) }) }
					t.Cleanup(func() {
						releaseClassification()
						releaseSecond()
						awaitPhaseTwoWorkers(t, &workers)
					})
					firstCallback, nextCallback := manager.BranchCommit, manager.BranchCommit
					if test.firstOperation == xa.PhaseTwoRollback {
						firstCallback = manager.BranchRollback
					}
					if test.nextOperation == xa.PhaseTwoRollback {
						nextCallback = manager.BranchRollback
					}
					firstDone := make(chan phaseTwoCallbackResult, 1)
					nextDone := make(chan phaseTwoCallbackResult, 1)
					workers.Add(1)
					go func() {
						defer workers.Done()
						status, err := firstCallback(context.Background(), branchResource)
						firstDone <- phaseTwoCallbackResult{status, err}
					}()
					select {
					case <-classifier.entered:
					case <-time.After(2 * time.Second):
						t.Fatal("first callback did not reach error classification")
					}

					waitContext := &phaseTwoWaitContext{Context: context.Background(), entered: make(chan struct{})}
					workers.Add(1)
					go func() {
						defer workers.Done()
						status, err := nextCallback(waitContext, branchResource)
						nextDone <- phaseTwoCallbackResult{status, err}
					}()
					select {
					case <-waitContext.entered:
					case <-resource.secondEntered:
						t.Error("another XA operation started before terminal classification completed")
					case <-time.After(2 * time.Second):
						t.Fatal("next callback did not reach the in-flight wait")
					}

					releaseClassification()
					first := awaitPhaseTwoResult(t, firstDone)
					assert.Equal(t, test.firstStatus, first.status)
					if test.firstErr {
						assert.ErrorIs(t, first.err, operationErr)
					} else {
						assert.NoError(t, first.err)
					}
					assert.Equal(t, int32(0), atomic.LoadInt32(&physical.closedDuringIO))
					releaseSecond()
					next := awaitPhaseTwoResult(t, nextDone)
					assert.Equal(t, test.nextStatus, next.status)
					if test.nextErr {
						assert.Error(t, next.err)
					} else {
						assert.NoError(t, next.err)
					}
					assert.Equal(t, int32(1), atomic.LoadInt32(&resource.calls))
					assert.Equal(t, test.firstOperation, classifier.classifiedAs)
					assertPhaseTwoConnectionHeld(t, manager, branchResource, xaID, false)
					wantCloses := int32(0)
					if discarded {
						wantCloses = 1
					}
					assert.Equal(t, wantCloses, atomic.LoadInt32(&physical.closeCalls))
				})
			}
		})
	}
}

func TestXAResourceManager_StaleTerminalCallbackPreservesReusedConnection(t *testing.T) {
	for _, operation := range []xa.PhaseTwoOperation{xa.PhaseTwoCommit, xa.PhaseTwoRollback} {
		name := "commit"
		if operation == xa.PhaseTwoRollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			connection, registrationManager := newMockXAConn(t, controller, 123)
			registrationManager.EXPECT().BranchReport(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			connection.res.keepInKeeper = true
			connection.res.shouldBeHeld = true
			t.Cleanup(func() { assert.NoError(t, connection.CloseForce()) })
			manager := &XAResourceManager{}
			manager.resourceCache.Store(connection.res.resourceID, connection.res)

			firstContext := tm.InitSeataContext(context.Background())
			tm.SetXID(firstContext, "branch-a")
			firstTx, err := connection.BeginTx(firstContext, driver.TxOptions{})
			require.NoError(t, err)
			require.NoError(t, firstTx.Commit())
			firstID := XaIdBuild("branch-a", 123)
			firstBranch := rm.BranchResource{
				BranchType: branch.BranchTypeXA, Xid: "branch-a", BranchId: 123, ResourceId: connection.res.resourceID,
			}
			staleOwner, err := manager.finishBranch(context.Background(), firstID, firstBranch)
			require.NoError(t, err)
			require.Same(t, connection, staleOwner)
			finishCallback := manager.BranchCommit
			staleCallback := staleOwner.xaCommitPhaseTwo
			wantStatus := branch.BranchStatus(branch.BranchStatusPhasetwoCommitted)
			if operation == xa.PhaseTwoRollback {
				finishCallback = manager.BranchRollback
				staleCallback = staleOwner.xaRollbackPhaseTwo
				wantStatus = branch.BranchStatusPhasetwoRollbacked
			}
			status, err := finishCallback(context.Background(), firstBranch)
			require.NoError(t, err)
			require.Equal(t, wantStatus, status)
			require.False(t, connection.isKept())

			branchStatusCache.Remove(firstID.String())
			status, err = staleCallback(context.Background(), firstID, nil)
			assert.ErrorIs(t, err, driver.ErrBadConn)
			assert.EqualValues(t, branch.BranchStatusUnknown, status)
			assert.False(t, connection.physicalClosed)
			setBranchStatus(firstID.String(), wantStatus)

			require.NoError(t, connection.ResetSession(context.Background()))
			secondContext := tm.InitSeataContext(context.Background())
			tm.SetXID(secondContext, "branch-b")
			secondTx, err := connection.BeginTx(secondContext, driver.TxOptions{})
			require.NoError(t, err)
			require.NoError(t, secondTx.Commit())
			secondID := XaIdBuild("branch-b", 123)
			secondOwner, held := connection.res.Lookup(secondID.String())
			require.True(t, held)
			require.Same(t, connection, secondOwner)
			require.True(t, connection.preparedForPhaseTwo)

			released := false
			status, err = staleCallback(context.Background(), firstID, func() { released = true })
			assert.NoError(t, err)
			assert.Equal(t, wantStatus, status)
			assert.False(t, released)
			secondOwner, held = connection.res.Lookup(secondID.String())
			assert.True(t, held, "a delayed callback for branch A released branch B's owner")
			assert.Same(t, connection, secondOwner)
			assert.True(t, connection.preparedForPhaseTwo)
			assert.False(t, connection.physicalClosed)

			branchStatusCache.Remove(firstID.String())
			status, err = staleCallback(context.Background(), firstID, func() { released = true })
			assert.ErrorIs(t, err, driver.ErrBadConn)
			assert.EqualValues(t, branch.BranchStatusUnknown, status)
			assert.False(t, released)
			secondOwner, held = connection.res.Lookup(secondID.String())
			assert.True(t, held)
			assert.Same(t, connection, secondOwner)
			assert.True(t, connection.preparedForPhaseTwo)
			assert.False(t, connection.physicalClosed)
		})
	}
}

func TestXAResourceManager_CanceledPhaseTwoWaiterPreservesOwner(t *testing.T) {
	for _, operation := range []xa.PhaseTwoOperation{xa.PhaseTwoCommit, xa.PhaseTwoRollback} {
		name := "waiting for commit"
		if operation == xa.PhaseTwoRollback {
			name = "waiting for rollback"
		}
		t.Run(name, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			resource := &blockingPhaseTwoTestXAResource{
				commitEntered: entered, releaseCommit: release, rollbackEntered: entered, releaseRollback: release,
			}
			manager, branchResource, xaID := newPhaseTwoTestManager(t, resource)
			cachedResource, found := manager.resourceCache.Load(branchResource.ResourceId)
			require.True(t, found)
			cachedConn, held := cachedResource.(*DBResource).Lookup(xaID)
			require.True(t, held)
			connection := cachedConn.(*XAConn)
			physical := &closeErrorDriverConn{}
			connection.targetConn = physical
			connection.poolDiscarded = true
			firstCallback, waitingCallback := manager.BranchCommit, manager.BranchRollback
			wantFirstStatus := branch.BranchStatus(branch.BranchStatusPhasetwoCommitted)
			wantWaitingStatus := branch.BranchStatus(branch.BranchStatusPhasetwoRollbackFailedRetryable)
			if operation == xa.PhaseTwoRollback {
				firstCallback, waitingCallback = manager.BranchRollback, manager.BranchCommit
				wantFirstStatus = branch.BranchStatusPhasetwoRollbacked
				wantWaitingStatus = branch.BranchStatusPhasetwoCommitFailedRetryable
			}
			cancelContext, cancel := context.WithCancel(context.Background())
			defer cancel()
			waitContext := &phaseTwoWaitContext{Context: cancelContext, entered: make(chan struct{})}
			var workers sync.WaitGroup
			var releaseOnce sync.Once
			releaseOperation := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(func() {
				cancel()
				releaseOperation()
				awaitPhaseTwoWorkers(t, &workers)
			})
			firstDone, waitingDone := make(chan phaseTwoCallbackResult, 1), make(chan phaseTwoCallbackResult, 1)
			workers.Add(1)
			go func() {
				defer workers.Done()
				status, err := firstCallback(context.Background(), branchResource)
				firstDone <- phaseTwoCallbackResult{status, err}
			}()
			select {
			case <-entered:
			case <-time.After(2 * time.Second):
				t.Fatal("first callback did not start its XA operation")
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				status, err := waitingCallback(waitContext, branchResource)
				waitingDone <- phaseTwoCallbackResult{status, err}
			}()
			select {
			case <-waitContext.entered:
			case <-time.After(2 * time.Second):
				t.Fatal("second callback did not reach the in-flight wait")
			}
			cancel()
			waiting := awaitPhaseTwoResult(t, waitingDone)
			assert.ErrorIs(t, waiting.err, context.Canceled)
			assert.Equal(t, wantWaitingStatus, waiting.status)
			assert.Equal(t, int32(0), atomic.LoadInt32(&physical.closeCalls))
			assertPhaseTwoConnectionHeld(t, manager, branchResource, xaID, true)
			cached, err := branchStatus(xaID)
			assert.NoError(t, err)
			assert.EqualValues(t, branch.BranchStatusUnknown, cached)

			releaseOperation()
			first := awaitPhaseTwoResult(t, firstDone)
			assert.NoError(t, first.err)
			assert.Equal(t, wantFirstStatus, first.status)
			assert.Equal(t, int32(1), atomic.LoadInt32(&physical.closeCalls))
			assert.Equal(t, 1, resource.commitCalls+resource.rollbackCalls)
			assertPhaseTwoConnectionHeld(t, manager, branchResource, xaID, false)
		})
	}
}

func TestXAResourceManager_FinishBranchErrorUsesCorrectDirection(t *testing.T) {
	branchStatusCache = gcache.New(16).LRU().Expiration(time.Minute).Build()
	manager := &XAResourceManager{}
	resource := rm.BranchResource{
		BranchType: branch.BranchTypeXA,
		Xid:        "127.0.0.1:8091:1001",
		BranchId:   2001,
		ResourceId: "missing-resource",
	}

	commitStatus, commitErr := manager.BranchCommit(context.Background(), resource)
	assert.Error(t, commitErr)
	assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedRetryable, commitStatus)

	rollbackStatus, rollbackErr := manager.BranchRollback(context.Background(), resource)
	assert.Error(t, rollbackErr)
	assert.EqualValues(t, branch.BranchStatusPhasetwoRollbackFailedRetryable, rollbackStatus)
}

type mysqlXAIntegrationSession struct {
	version      string
	connectionID uint64
	detach       int
}

type mysqlXAIntegrationConnector struct {
	driver.Connector
	mu       sync.Mutex
	sessions []mysqlXAIntegrationSession
}

func (connector *mysqlXAIntegrationConnector) Connect(ctx context.Context) (driver.Conn, error) {
	connection, err := connector.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := connection.(driver.QueryerContext).QueryContext(ctx,
		"SELECT VERSION(), CONNECTION_ID(), @@session.xa_detach_on_prepare", nil)
	if err != nil {
		connection.Close()
		return nil, err
	}
	values := make([]driver.Value, 3)
	err = rows.Next(values)
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		connection.Close()
		if err != nil {
			return nil, err
		}
		return nil, closeErr
	}
	text := func(value driver.Value) string {
		if bytes, ok := value.([]byte); ok {
			return string(bytes)
		}
		return fmt.Sprint(value)
	}
	connectionID, err := strconv.ParseUint(text(values[1]), 10, 64)
	if err != nil {
		connection.Close()
		return nil, err
	}
	detach, err := strconv.Atoi(text(values[2]))
	if err != nil {
		connection.Close()
		return nil, err
	}
	connector.mu.Lock()
	connector.sessions = append(connector.sessions, mysqlXAIntegrationSession{
		version: text(values[0]), connectionID: connectionID, detach: detach,
	})
	connector.mu.Unlock()
	return connection, nil
}

func (connector *mysqlXAIntegrationConnector) snapshot() []mysqlXAIntegrationSession {
	connector.mu.Lock()
	defer connector.mu.Unlock()
	return append([]mysqlXAIntegrationSession(nil), connector.sessions...)
}

type mysqlXAIntegrationEnvironment struct {
	dsn          string
	admin        *gosql.DB
	observer     *gosql.DB
	table        string
	runID        string
	logStartedAt string
	nextRow      int64
}

type mysqlXAIntegrationBranch struct {
	resource rm.BranchResource
	xaID     *XABranchXid
	owner    *XAConn
}

type mysqlXAIntegrationFixture struct {
	pool     *gosql.DB
	resource *DBResource
	manager  *XAResourceManager
	recorder *mysqlXAIntegrationConnector
	branches []mysqlXAIntegrationBranch
	reports  []rm.BranchReportParam
	env      *mysqlXAIntegrationEnvironment
	mode     string
	nextID   int64
	ctx      context.Context
	cancel   context.CancelFunc
}

func TestMySQLXAIntegration(t *testing.T) {
	dsn, adminDSN := os.Getenv("SEATA_XA_MYSQL_DSN"), os.Getenv("SEATA_XA_MYSQL_ADMIN_DSN")
	if dsn == "" && adminDSN == "" {
		t.Skip("SEATA_XA_MYSQL_DSN and SEATA_XA_MYSQL_ADMIN_DSN are not set")
	}
	require.NotEmpty(t, dsn, "SEATA_XA_MYSQL_DSN must be set")
	require.NotEmpty(t, adminDSN, "SEATA_XA_MYSQL_ADMIN_DSN must be set")
	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	require.NotEmpty(t, config.DBName, "the XA DSN must select a dedicated test database")
	observer, err := gosql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, observer.Close()) })
	admin, err := gosql.Open("mysql", adminDSN)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, admin.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, observer.PingContext(ctx))
	require.NoError(t, admin.PingContext(ctx))
	var observerServer, adminServer, logOutput, logStartedAt string
	var generalLog int
	require.NoError(t, observer.QueryRowContext(ctx, "SELECT @@server_uuid").Scan(&observerServer))
	require.NoError(t, admin.QueryRowContext(ctx,
		"SELECT @@server_uuid, @@global.general_log, @@global.log_output, DATE_FORMAT(CURRENT_TIMESTAMP(6), '%Y-%m-%d %H:%i:%s.%f')").
		Scan(&adminServer, &generalLog, &logOutput, &logStartedAt))
	require.Equal(t, observerServer, adminServer, "both DSNs must point to the same MySQL instance")
	require.Equal(t, 1, generalLog, "the isolated instance must enable general_log")
	require.Contains(t, logOutput, "TABLE", "the isolated instance must log queries to mysql.general_log")
	_, err = mysqlXAIntegrationRecover(ctx, admin)
	require.NoError(t, err, "the admin connection must be permitted to run XA RECOVER")
	runID := strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	environment := &mysqlXAIntegrationEnvironment{
		dsn: dsn, admin: admin, observer: observer, table: "seata_xa_it_" + runID, runID: runID,
		logStartedAt: logStartedAt,
	}
	_, err = observer.ExecContext(ctx, "CREATE TABLE `"+environment.table+"` (id BIGINT PRIMARY KEY, value INT NOT NULL) ENGINE=InnoDB")
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, cleanupErr := observer.ExecContext(cleanupContext, "DROP TABLE IF EXISTS `"+environment.table+"`")
		assert.NoError(t, cleanupErr)
	})

	for _, mode := range []string{"ON", "OFF"} {
		t.Run(mode, func(t *testing.T) {
			for _, commit := range []bool{true, false} {
				action := "rollback"
				if commit {
					action = "commit"
				}
				t.Run("prepared_"+action, func(t *testing.T) {
					mysqlXAIntegrationPrepared(t, environment, mode, commit, false)
				})
				t.Run("timeout_"+action, func(t *testing.T) {
					mysqlXAIntegrationPrepared(t, environment, mode, commit, true)
				})
			}
			t.Run("consecutive_local_rollbacks", func(t *testing.T) {
				mysqlXAIntegrationPoolReuse(t, environment, mode)
			})
		})
	}
	t.Run("actual_session_overrides_resource_default", func(t *testing.T) {
		fixture := newMySQLXAIntegrationFixture(t, environment, "ON")
		require.False(t, fixture.resource.shouldBeHeld)
		connection, owner, session := fixture.checkout(t)
		_, err := connection.ExecContext(fixture.ctx, "SET SESSION xa_detach_on_prepare=OFF")
		require.NoError(t, err)
		var actual mysqlXAIntegrationSession
		require.NoError(t, connection.QueryRowContext(fixture.ctx,
			"SELECT VERSION(), CONNECTION_ID(), @@session.xa_detach_on_prepare").Scan(&actual.version, &actual.connectionID, &actual.detach))
		require.Equal(t, session.connectionID, actual.connectionID)
		require.Equal(t, 0, actual.detach)
		transaction, activeBranch, rowID := fixture.begin(t, connection, owner)
		require.True(t, owner.ShouldBeHeld(), "the actual OFF session must override the resource's ON default")
		require.NoError(t, transaction.Commit())
		fixture.assertRecovered(t, activeBranch.xaID.String(), true)
		require.NoError(t, connection.Close())
		replacement, _, replacementSession := fixture.checkout(t)
		require.NotEqual(t, session.connectionID, replacementSession.connectionID)
		require.NoError(t, replacement.Close())
		fixture.manager.closeTimedOutPhaseTwoConnections(time.Now().Add(2 * time.Second))
		require.True(t, owner.isKept())
		require.False(t, owner.physicalClosed)
		status, err := fixture.manager.BranchRollback(fixture.ctx, activeBranch.resource)
		require.NoError(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
		assert.Empty(t, fixture.recorder.snapshot())
		fixture.assertRow(t, rowID, 0)
		fixture.assertRecovered(t, activeBranch.xaID.String(), false)
		fixture.assertQueryLog(t, session.connectionID, "SELECT @@session.xa_detach_on_prepare")
		fixture.assertQueryLog(t, session.connectionID, "XA ROLLBACK '"+activeBranch.xaID.String()+"'")
		t.Logf("resource_default=ON actual_session=OFF xid=%s owner_connection_id=%d pool_replacement_connection_id=%d",
			activeBranch.xaID.String(), session.connectionID, replacementSession.connectionID)
	})
}

func newMySQLXAIntegrationFixture(t *testing.T, environment *mysqlXAIntegrationEnvironment, mode string) *mysqlXAIntegrationFixture {
	t.Helper()
	previousCache, previousTimeout := branchStatusCache, xaConnTimeout
	branchStatusCache = gcache.New(128).LRU().Build()
	xaConnTimeout = time.Minute
	t.Cleanup(func() { branchStatusCache, xaConnTimeout = previousCache, previousTimeout })
	fixture := &mysqlXAIntegrationFixture{
		env: environment, mode: mode, nextID: 1000,
		manager: &XAResourceManager{config: XAConfig{TwoPhaseHoldTime: time.Second}},
	}
	fixture.ctx, fixture.cancel = context.WithTimeout(context.Background(), 30*time.Second)
	controller := gomock.NewController(t)
	manager := mock.NewMockDataSourceManager(controller)
	manager.SetBranchType(branch.BranchTypeXA)
	registerResourceManagerForTest(t, manager)
	manager.EXPECT().RegisterResource(gomock.Any()).DoAndReturn(func(resource rm.Resource) error {
		fixture.resource = resource.(*DBResource)
		fixture.manager.resourceCache.Store(resource.GetResourceId(), resource)
		return nil
	}).Times(1)
	manager.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, request rm.BranchRegisterParam) (int64, error) {
		assert.Equal(t, branch.BranchTypeXA, request.BranchType)
		fixture.nextID++
		return fixture.nextID, nil
	}).AnyTimes()
	manager.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, report rm.BranchReportParam) error {
		fixture.reports = append(fixture.reports, report)
		return nil
	}).AnyTimes()
	config, err := mysql.ParseDSN(environment.dsn)
	require.NoError(t, err)
	if config.Params == nil {
		config.Params = make(map[string]string)
	}
	config.Params["xa_detach_on_prepare"] = mode
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 10 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}
	fixture.pool, err = gosql.Open(SeataXAMySQLDriver, config.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { fixture.cleanup(t) })
	require.NotNil(t, fixture.resource)
	fixture.pool.SetMaxOpenConns(1)
	fixture.pool.SetMaxIdleConns(1)
	fixture.recorder = &mysqlXAIntegrationConnector{Connector: fixture.resource.connector}
	fixture.resource.connector = fixture.recorder
	return fixture
}

func (fixture *mysqlXAIntegrationFixture) checkout(t *testing.T) (*gosql.Conn, *XAConn, mysqlXAIntegrationSession) {
	t.Helper()
	connection, err := fixture.pool.Conn(fixture.ctx)
	require.NoError(t, err)
	t.Cleanup(func() { connection.Close() })
	var session mysqlXAIntegrationSession
	require.NoError(t, connection.QueryRowContext(fixture.ctx,
		"SELECT VERSION(), CONNECTION_ID(), @@session.xa_detach_on_prepare").Scan(&session.version, &session.connectionID, &session.detach))
	fixture.assertSession(t, session)
	var owner *XAConn
	require.NoError(t, connection.Raw(func(raw interface{}) error {
		var ok bool
		owner, ok = raw.(*XAConn)
		if !ok {
			return fmt.Errorf("expected an XAConn, got %T", raw)
		}
		return nil
	}))
	return connection, owner, session
}

func (fixture *mysqlXAIntegrationFixture) assertSession(t *testing.T, session mysqlXAIntegrationSession) {
	t.Helper()
	version, err := util.ConvertDbVersion(session.version)
	require.NoError(t, err)
	minimum, err := util.ConvertDbVersion("8.0.29")
	require.NoError(t, err)
	require.GreaterOrEqual(t, version, minimum)
	require.NotContains(t, strings.ToLower(session.version), "mariadb")
	wantDetach := 0
	if fixture.mode == "ON" {
		wantDetach = 1
	}
	require.Equal(t, wantDetach, session.detach, "effective session xa_detach_on_prepare differs from the test mode")
	t.Logf("mysql_version=%s mode=%s connection_id=%d xa_detach_on_prepare=%d", session.version, fixture.mode, session.connectionID, session.detach)
}

func (fixture *mysqlXAIntegrationFixture) begin(t *testing.T, connection *gosql.Conn, owner *XAConn) (*gosql.Tx, mysqlXAIntegrationBranch, int64) {
	t.Helper()
	fixture.env.nextRow++
	rowID := fixture.env.nextRow
	globalXID := fmt.Sprintf("xa-it-%s-%d", fixture.env.runID, rowID)
	globalContext := tm.InitSeataContext(fixture.ctx)
	tm.SetXID(globalContext, globalXID)
	transaction, err := connection.BeginTx(globalContext, nil)
	require.NoError(t, err)
	t.Cleanup(func() { transaction.Rollback() })
	activeBranch := mysqlXAIntegrationBranch{
		owner: owner, xaID: owner.xaBranchXid,
		resource: rm.BranchResource{
			BranchType: branch.BranchTypeXA, Xid: globalXID, BranchId: int64(owner.xaBranchXid.GetBranchId()), ResourceId: fixture.resource.resourceID,
		},
	}
	fixture.branches = append(fixture.branches, activeBranch)
	_, err = transaction.ExecContext(fixture.ctx,
		fmt.Sprintf("INSERT INTO `%s` (id, value) VALUES (%d, 1)", fixture.env.table, rowID))
	require.NoError(t, err)
	return transaction, activeBranch, rowID
}

func (fixture *mysqlXAIntegrationFixture) assertRow(t *testing.T, rowID int64, want int) {
	t.Helper()
	var count int
	require.NoError(t, fixture.env.observer.QueryRowContext(fixture.ctx,
		"SELECT COUNT(*) FROM `"+fixture.env.table+"` WHERE id = ?", rowID).Scan(&count))
	assert.Equal(t, want, count)
}

func (fixture *mysqlXAIntegrationFixture) assertRecovered(t *testing.T, xaID string, want bool) {
	t.Helper()
	recovered, err := mysqlXAIntegrationRecover(fixture.ctx, fixture.env.admin)
	require.NoError(t, err)
	assert.Equal(t, want, recovered[xaID], "XA RECOVER state for %s", xaID)
}

func (fixture *mysqlXAIntegrationFixture) assertQueryLog(t *testing.T, connectionID uint64, expected ...string) {
	t.Helper()
	placeholders := make([]string, len(expected))
	arguments := []interface{}{fixture.env.logStartedAt, connectionID}
	for index, query := range expected {
		placeholders[index] = "?"
		arguments = append(arguments, query)
	}
	rows, err := fixture.env.admin.QueryContext(fixture.ctx,
		"SELECT argument FROM mysql.general_log WHERE event_time >= ? AND thread_id = ? AND command_type = 'Query' AND argument IN ("+
			strings.Join(placeholders, ",")+") ORDER BY event_time", arguments...)
	require.NoError(t, err)
	defer rows.Close()
	var actual []string
	for rows.Next() {
		var query string
		require.NoError(t, rows.Scan(&query))
		actual = append(actual, query)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, expected, actual, "unexpected XA query sequence on physical connection %d", connectionID)
}

func mysqlXAIntegrationPrepared(t *testing.T, environment *mysqlXAIntegrationEnvironment, mode string, commit, timeout bool) {
	fixture := newMySQLXAIntegrationFixture(t, environment, mode)
	connection, owner, session := fixture.checkout(t)
	transaction, activeBranch, rowID := fixture.begin(t, connection, owner)
	require.NoError(t, transaction.Commit())
	require.Len(t, fixture.reports, 1)
	assert.EqualValues(t, branch.BranchStatusPhaseoneDone, fixture.reports[0].Status)
	fixture.assertRow(t, rowID, 0)
	fixture.assertRecovered(t, activeBranch.xaID.String(), true)
	assert.True(t, owner.isKept())
	require.NoError(t, connection.Close())
	replacement, replacementOwner, replacementSession := fixture.checkout(t)
	assert.NotEqual(t, session.connectionID, replacementSession.connectionID)
	assert.NotSame(t, owner, replacementOwner)
	assert.True(t, owner.isKept())
	assert.False(t, owner.physicalClosed)
	assert.True(t, owner.poolDiscarded)
	require.NoError(t, replacement.Close())
	if timeout {
		fixture.manager.closeTimedOutPhaseTwoConnections(time.Now().Add(2 * time.Second))
		_, held := fixture.resource.Lookup(activeBranch.xaID.String())
		assert.Equal(t, mode == "OFF", held)
		assert.Equal(t, mode == "ON", owner.physicalClosed)
		fixture.assertRecovered(t, activeBranch.xaID.String(), true)
	}
	callback, opposite := fixture.manager.BranchRollback, fixture.manager.BranchCommit
	wantStatus := branch.BranchStatus(branch.BranchStatusPhasetwoRollbacked)
	wantOppositeStatus := branch.BranchStatus(branch.BranchStatusPhasetwoCommitFailedUnretryable)
	wantRows, command := 0, "XA ROLLBACK "
	if commit {
		callback, opposite = fixture.manager.BranchCommit, fixture.manager.BranchRollback
		wantStatus = branch.BranchStatusPhasetwoCommitted
		wantOppositeStatus = branch.BranchStatusPhasetwoRollbackFailedUnretryable
		wantRows, command = 1, "XA COMMIT "
	}
	status, err := callback(fixture.ctx, activeBranch.resource)
	require.NoError(t, err)
	assert.Equal(t, wantStatus, status)
	status, err = callback(fixture.ctx, activeBranch.resource)
	require.NoError(t, err)
	assert.Equal(t, wantStatus, status)
	status, err = opposite(fixture.ctx, activeBranch.resource)
	assert.Error(t, err)
	assert.Equal(t, wantOppositeStatus, status)
	fixture.assertRow(t, rowID, wantRows)
	fixture.assertRecovered(t, activeBranch.xaID.String(), false)
	assert.False(t, owner.isKept())
	assert.True(t, owner.physicalClosed)
	phaseTwoConnectionID := session.connectionID
	recoverySessions := fixture.recorder.snapshot()
	if timeout && mode == "ON" {
		require.Len(t, recoverySessions, 1)
		fixture.assertSession(t, recoverySessions[0])
		phaseTwoConnectionID = recoverySessions[0].connectionID
		assert.NotEqual(t, session.connectionID, phaseTwoConnectionID)
		assert.NotEqual(t, replacementSession.connectionID, phaseTwoConnectionID)
	} else {
		assert.Empty(t, recoverySessions)
	}
	fixture.assertQueryLog(t, phaseTwoConnectionID, command+"'"+activeBranch.xaID.String()+"'")
	t.Logf("xid=%s owner_connection_id=%d pool_replacement_connection_id=%d phase_two_connection_id=%d timeout=%t commit=%t",
		activeBranch.xaID.String(), session.connectionID, replacementSession.connectionID, phaseTwoConnectionID, timeout, commit)
}

func mysqlXAIntegrationPoolReuse(t *testing.T, environment *mysqlXAIntegrationEnvironment, mode string) {
	fixture := newMySQLXAIntegrationFixture(t, environment, mode)
	var previousSession mysqlXAIntegrationSession
	var previousOwner *XAConn
	for iteration := 0; iteration < 2; iteration++ {
		connection, owner, session := fixture.checkout(t)
		if iteration > 0 {
			assert.Equal(t, previousSession.connectionID, session.connectionID, "successful local rollback must reuse the physical connection")
			assert.Same(t, previousOwner, owner)
			assert.False(t, owner.rollBacked, "database/sql checkout must reset the previous rollback flag")
		}
		transaction, activeBranch, rowID := fixture.begin(t, connection, owner)
		require.NoError(t, transaction.Rollback())
		assert.False(t, owner.isKept())
		assert.False(t, owner.physicalClosed)
		require.NoError(t, connection.Close())
		fixture.assertRow(t, rowID, 0)
		fixture.assertRecovered(t, activeBranch.xaID.String(), false)
		status, err := fixture.manager.BranchRollback(fixture.ctx, activeBranch.resource)
		require.NoError(t, err, "TC rollback must accept real MySQL XAER_NOTA after local rollback")
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
		status, err = fixture.manager.BranchRollback(fixture.ctx, activeBranch.resource)
		require.NoError(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
		status, err = fixture.manager.BranchCommit(fixture.ctx, activeBranch.resource)
		assert.Error(t, err)
		assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedUnretryable, status)
		fixture.assertQueryLog(t, session.connectionID,
			"XA END '"+activeBranch.xaID.String()+"'", "XA ROLLBACK '"+activeBranch.xaID.String()+"'")
		t.Logf("local_rollback=%d xid=%s owner_connection_id=%d", iteration+1, activeBranch.xaID.String(), session.connectionID)
		previousSession, previousOwner = session, owner
	}
	require.Len(t, fixture.reports, 2)
	for _, report := range fixture.reports {
		assert.EqualValues(t, branch.BranchStatusPhaseoneFailed, report.Status)
	}
	assert.Len(t, fixture.recorder.snapshot(), 2)
}

func mysqlXAIntegrationRecover(ctx context.Context, admin *gosql.DB) (map[string]bool, error) {
	rows, err := admin.QueryContext(ctx, "XA RECOVER")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recovered := make(map[string]bool)
	for rows.Next() {
		var formatID, globalLength, branchLength int
		var data []byte
		if err := rows.Scan(&formatID, &globalLength, &branchLength, &data); err != nil {
			return nil, err
		}
		if globalLength+branchLength != len(data) {
			return nil, fmt.Errorf("unexpected XA RECOVER data length: %d != %d", globalLength+branchLength, len(data))
		}
		recovered[string(data)] = true
	}
	return recovered, rows.Err()
}

func (fixture *mysqlXAIntegrationFixture) cleanup(t *testing.T) {
	t.Helper()
	defer fixture.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, activeBranch := range fixture.branches {
		owner := activeBranch.owner
		owner.keeperMu.Lock()
		matches := !owner.physicalClosed && owner.xaBranchXid != nil && owner.xaBranchXid.String() == activeBranch.xaID.String()
		resource := owner.xaResource
		owner.keeperMu.Unlock()
		if matches && resource != nil {
			resource.End(ctx, activeBranch.xaID.String(), xa.TMFail)
			owner.XaRollback(ctx, activeBranch.xaID)
		}
	}
	assert.NoError(t, fixture.pool.Close())
	for _, activeBranch := range fixture.branches {
		assert.NoError(t, activeBranch.owner.CloseForce())
	}
	if fixture.resource != nil {
		assert.NoError(t, fixture.resource.db.Close())
		assert.NoError(t, fixture.resource.metaCache.Destroy())
	}
	recovered, err := mysqlXAIntegrationRecover(ctx, fixture.env.admin)
	if !assert.NoError(t, err) {
		return
	}
	for _, activeBranch := range fixture.branches {
		if recovered[activeBranch.xaID.String()] {
			_, err := fixture.env.admin.ExecContext(ctx, "XA ROLLBACK '"+activeBranch.xaID.String()+"'")
			assert.NoError(t, err, "cleanup of prepared XA branch %s", activeBranch.xaID.String())
		}
	}
	recovered, err = mysqlXAIntegrationRecover(ctx, fixture.env.admin)
	if assert.NoError(t, err) {
		for _, activeBranch := range fixture.branches {
			assert.False(t, recovered[activeBranch.xaID.String()], "XA branch remains after cleanup: %s", activeBranch.xaID.String())
		}
	}
}
