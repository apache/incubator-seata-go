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
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/bluele/gcache"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
	gettyrm "seata.apache.org/seata-go/v2/pkg/rm/remoting/getty"
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
	commitErr         error
	rollbackErr       error
	alreadyCommitted  bool
	alreadyRollbacked bool
	unretryable       bool
	commitCalls       int
	rollbackCalls     int
}

func (r *phaseTwoTestXAResource) Commit(ctx context.Context, xid string, onePhase bool) error {
	r.commitCalls++
	return r.commitErr
}

func (r *phaseTwoTestXAResource) End(ctx context.Context, xid string, flags int) error {
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

func (r *phaseTwoTestXAResource) IsAlreadyCommitted(err error) bool {
	return r.alreadyCommitted
}

func (r *phaseTwoTestXAResource) IsAlreadyRollbacked(err error) bool {
	return r.alreadyRollbacked
}

func (r *phaseTwoTestXAResource) IsUnretryable(err error) bool {
	return r.unretryable
}

func newPhaseTwoTestManager(
	t *testing.T,
	resource *phaseTwoTestXAResource,
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
	xaID := XaIdBuild(globalXID, uint64(branchID))
	xaConn := &XAConn{
		Conn: &Conn{
			targetConn: &phaseTwoTestDriverConn{},
			res:        dbResource,
			dbType:     types.DBTypeMySQL,
		},
		xaResource:        resource,
		xaErrorClassifier: resource,
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
			commitErr:        errors.New("already committed"),
			alreadyCommitted: true,
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
			commitErr:         commitErr,
			alreadyRollbacked: true,
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
			commitErr:   commitErr,
			unretryable: true,
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
			rollbackErr:       errors.New("already rollbacked"),
			alreadyRollbacked: true,
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
			rollbackErr:      rollbackErr,
			alreadyCommitted: true,
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
