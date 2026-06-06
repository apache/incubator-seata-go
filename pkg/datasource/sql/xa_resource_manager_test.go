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
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	xares "seata.apache.org/seata-go/v2/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

type fakeXAResource struct {
	commitXIDs   []string
	rollbackXIDs []string
	endFlags     []int
	commitErr    error
	endErr       error
	prepareErr   error
	rollbackErr  error
}

func (f *fakeXAResource) Commit(_ context.Context, xid string, _ bool) error {
	f.commitXIDs = append(f.commitXIDs, xid)
	return f.commitErr
}

func (f *fakeXAResource) End(_ context.Context, _ string, flags int) error {
	f.endFlags = append(f.endFlags, flags)
	return f.endErr
}

func (f *fakeXAResource) Forget(context.Context, string) error {
	return nil
}

func (f *fakeXAResource) GetTransactionTimeout() time.Duration {
	return 0
}

func (f *fakeXAResource) IsSameRM(context.Context, xares.XAResource) bool {
	return false
}

func (f *fakeXAResource) XAPrepare(context.Context, string) error {
	return f.prepareErr
}

func (f *fakeXAResource) Recover(context.Context, int) ([]string, error) {
	return nil, nil
}

func (f *fakeXAResource) Rollback(_ context.Context, xid string) error {
	f.rollbackXIDs = append(f.rollbackXIDs, xid)
	return f.rollbackErr
}

func (f *fakeXAResource) SetTransactionTimeout(time.Duration) bool {
	return false
}

func (f *fakeXAResource) Start(context.Context, string, int) error {
	return nil
}

type closeTrackingConn struct {
	closeCalls int
}

func (c *closeTrackingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *closeTrackingConn) Close() error {
	c.closeCalls++
	return nil
}

func (c *closeTrackingConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
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

			xaManager := &XAResourceManager{rmRemoting: rm.GetRMRemotingInstance()}

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

func TestDBResource_ConnectionForXA_UsesResourceDBType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().Close().AnyTimes().Return(nil)

	mockConnector := mock.NewMockTestDriverConnector(ctrl)
	mockConnector.EXPECT().Connect(gomock.Any()).Return(mockConn, nil)

	resource := &DBResource{
		connector: mockConnector,
		dbType:    types.DBTypeOracle,
	}

	xaConn, err := resource.ConnectionForXA(context.Background(), XaIdBuild("global-xid", 9))
	require.NoError(t, err)
	require.NotNil(t, xaConn)
	assert.IsType(t, &xares.OracleXAConn{}, xaConn.xaResource)
	assert.IsType(t, &xares.OracleXAErrorClassifier{}, xaConn.xaErrorClassifier)
	assert.Equal(t, "global-xid-9", xaConn.xaBranchXid.String())
}

func TestDBResource_ConnectionForXA_ReturnsHeldConnectionForRetry(t *testing.T) {
	resource := &DBResource{}
	held := &XAConn{}
	xaID := XaIdBuild("held-xid", 7)

	require.NoError(t, resource.Hold(xaID.String(), held))

	got, err := resource.ConnectionForXA(context.Background(), xaID)
	require.NoError(t, err)
	assert.Same(t, held, got)
}

func TestDBResource_InitMarksOracleAsHeld(t *testing.T) {
	resource, err := newResource(withDBType(types.DBTypeOracle))
	require.NoError(t, err)
	assert.True(t, resource.IsShouldBeHeld())
}

func TestXAResourceManager_BranchCommit_UsesMappedXidAndReleasesHeldConnection(t *testing.T) {
	xaID := XaIdBuild("commit-xid", 21)
	fakeResource := &fakeXAResource{}
	targetConn := &closeTrackingConn{}
	dbResource := &DBResource{
		resourceID:   "resource-commit",
		dbType:       types.DBTypeMySQL,
		shouldBeHeld: true,
	}

	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
		},
		xaBranchXid: xaID,
		xaResource:  fakeResource,
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	status, err := manager.BranchCommit(context.Background(), rm.BranchResource{
		ResourceId: dbResource.resourceID,
		Xid:        xaID.GetGlobalXid(),
		BranchId:   int64(xaID.GetBranchId()),
	})
	require.NoError(t, err)
	assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, status)
	assert.Equal(t, []string{xaID.String()}, fakeResource.commitXIDs)
	_, ok := dbResource.Lookup(xaID.String())
	assert.False(t, ok)
	assert.Equal(t, 1, targetConn.closeCalls)
}

func TestXAResourceManager_BranchCommit_ReturnsCommitFailureWhenFinishBranchFails(t *testing.T) {
	manager := &XAResourceManager{}

	status, err := manager.BranchCommit(context.Background(), rm.BranchResource{
		ResourceId: "missing-resource",
		Xid:        "commit-xid",
		BranchId:   23,
	})

	require.Error(t, err)
	assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedRetryable, status)
}

func TestXAResourceManager_BranchCommit_PreservesHeldConnectionOnError(t *testing.T) {
	xaID := XaIdBuild("commit-error-xid", 23)
	commitErr := errors.New("temporary commit failure")
	fakeResource := &fakeXAResource{commitErr: commitErr}
	targetConn := &closeTrackingConn{}
	dbResource := &DBResource{
		resourceID:   "resource-commit-error",
		dbType:       types.DBTypeMySQL,
		shouldBeHeld: true,
	}

	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
		},
		xaBranchXid: xaID,
		xaResource:  fakeResource,
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	status, err := manager.BranchCommit(context.Background(), rm.BranchResource{
		ResourceId: dbResource.resourceID,
		Xid:        xaID.GetGlobalXid(),
		BranchId:   int64(xaID.GetBranchId()),
	})
	require.ErrorIs(t, err, commitErr)
	assert.EqualValues(t, branch.BranchStatusPhasetwoCommitFailedRetryable, status)
	assert.Equal(t, []string{xaID.String()}, fakeResource.commitXIDs)
	held, ok := dbResource.Lookup(xaID.String())
	require.True(t, ok)
	assert.Same(t, heldConn, held)
	assert.Equal(t, 0, targetConn.closeCalls)
}

func TestXAResourceManager_BranchCommit_IgnoresTargetCommitErrorAfterXACommit(t *testing.T) {
	xaID := XaIdBuild("commit-target-error-xid", 30)
	targetErr := errors.New("target commit failed")
	fakeResource := &fakeXAResource{}
	targetConn := &closeTrackingConn{}
	targetTx := &countingDriverTx{commitErr: targetErr}
	dbResource := &DBResource{
		resourceID:   "resource-commit-target-error",
		dbType:       types.DBTypeOracle,
		shouldBeHeld: true,
	}

	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
		},
		tx:          &Tx{target: targetTx},
		xaBranchXid: xaID,
		xaResource:  fakeResource,
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	status, err := manager.BranchCommit(context.Background(), rm.BranchResource{
		ResourceId: dbResource.resourceID,
		Xid:        xaID.GetGlobalXid(),
		BranchId:   int64(xaID.GetBranchId()),
	})
	require.NoError(t, err)
	assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, status)
	_, ok := dbResource.Lookup(xaID.String())
	assert.False(t, ok)
	assert.Equal(t, 1, targetTx.commitCalls)
	assert.Equal(t, 0, targetTx.rollbackCalls)
	assert.Equal(t, 1, targetConn.closeCalls)
}

func TestXAResourceManager_BranchRollback_UsesMappedXidAndReleasesHeldConnection(t *testing.T) {
	xaID := XaIdBuild("rollback-xid", 22)
	fakeResource := &fakeXAResource{}
	targetConn := &closeTrackingConn{}
	dbResource := &DBResource{
		resourceID:   "resource-rollback",
		dbType:       types.DBTypeMySQL,
		shouldBeHeld: true,
	}

	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
		},
		xaBranchXid: xaID,
		xaResource:  fakeResource,
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	status, err := manager.BranchRollback(context.Background(), rm.BranchResource{
		ResourceId: dbResource.resourceID,
		Xid:        xaID.GetGlobalXid(),
		BranchId:   int64(xaID.GetBranchId()),
	})
	require.NoError(t, err)
	assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
	assert.Equal(t, []string{xaID.String()}, fakeResource.rollbackXIDs)
	_, ok := dbResource.Lookup(xaID.String())
	assert.False(t, ok)
	assert.Equal(t, 1, targetConn.closeCalls)
}

func TestXAResourceManager_BranchRollback_PreservesHeldConnectionOnError(t *testing.T) {
	xaID := XaIdBuild("rollback-error-xid", 24)
	rollbackErr := errors.New("temporary rollback failure")
	fakeResource := &fakeXAResource{rollbackErr: rollbackErr}
	targetConn := &closeTrackingConn{}
	dbResource := &DBResource{
		resourceID:   "resource-rollback-error",
		dbType:       types.DBTypeMySQL,
		shouldBeHeld: true,
	}

	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
		},
		xaBranchXid: xaID,
		xaResource:  fakeResource,
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	status, err := manager.BranchRollback(context.Background(), rm.BranchResource{
		ResourceId: dbResource.resourceID,
		Xid:        xaID.GetGlobalXid(),
		BranchId:   int64(xaID.GetBranchId()),
	})
	require.ErrorIs(t, err, rollbackErr)
	assert.EqualValues(t, branch.BranchStatusPhasetwoRollbackFailedRetryable, status)
	assert.Equal(t, []string{xaID.String()}, fakeResource.rollbackXIDs)
	held, ok := dbResource.Lookup(xaID.String())
	require.True(t, ok)
	assert.Same(t, heldConn, held)
	assert.Equal(t, 0, targetConn.closeCalls)
}

func TestXAResourceManager_BranchRollback_IgnoresTargetRollbackErrorAfterXARollback(t *testing.T) {
	xaID := XaIdBuild("rollback-target-error-xid", 34)
	targetErr := errors.New("target rollback failed")
	fakeResource := &fakeXAResource{}
	targetConn := &closeTrackingConn{}
	targetTx := &countingDriverTx{rollbackErr: targetErr}
	dbResource := &DBResource{
		resourceID:   "resource-rollback-target-error",
		dbType:       types.DBTypeOracle,
		shouldBeHeld: true,
	}

	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
		},
		tx:          &Tx{target: targetTx},
		xaBranchXid: xaID,
		xaResource:  fakeResource,
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	status, err := manager.BranchRollback(context.Background(), rm.BranchResource{
		ResourceId: dbResource.resourceID,
		Xid:        xaID.GetGlobalXid(),
		BranchId:   int64(xaID.GetBranchId()),
	})
	require.NoError(t, err)
	assert.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
	_, ok := dbResource.Lookup(xaID.String())
	assert.False(t, ok)
	assert.Equal(t, 0, targetTx.commitCalls)
	assert.Equal(t, 1, targetTx.rollbackCalls)
	assert.Equal(t, 1, targetConn.closeCalls)
}

func TestXAResourceManager_CloseTimeoutXAConnections_SkipsHeldOraclePreparedConnection(t *testing.T) {
	xaID := XaIdBuild("timeout-xid", 31)
	targetConn := &closeTrackingConn{}
	dbResource := &DBResource{
		resourceID:   "resource-timeout",
		dbType:       types.DBTypeOracle,
		shouldBeHeld: true,
	}
	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
			txCtx: &types.TransactionContext{
				XID: xaID.GetGlobalXid(),
			},
		},
		xaBranchXid: xaID,
		prepareTime: time.Now().Add(-2 * time.Second),
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{
		config: XAConfig{TwoPhaseHoldTime: time.Second},
	}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	manager.closeTimeoutXAConnections(time.Now())

	held, ok := dbResource.Lookup(xaID.String())
	require.True(t, ok)
	assert.Same(t, heldConn, held)
	assert.Equal(t, 0, targetConn.closeCalls)
}

func TestXAResourceManager_CloseTimeoutXAConnections_IgnoresUnpreparedHeldConnection(t *testing.T) {
	xaID := XaIdBuild("active-xid", 32)
	targetConn := &closeTrackingConn{}
	dbResource := &DBResource{
		resourceID:   "resource-active",
		dbType:       types.DBTypeOracle,
		shouldBeHeld: true,
	}
	heldConn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        dbResource,
			txCtx: &types.TransactionContext{
				XID: xaID.GetGlobalXid(),
			},
		},
		xaBranchXid: xaID,
		isConnKept:  true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), heldConn))

	manager := &XAResourceManager{
		config: XAConfig{TwoPhaseHoldTime: time.Second},
	}
	manager.resourceCache.Store(dbResource.resourceID, dbResource)

	manager.closeTimeoutXAConnections(time.Now().Add(time.Hour))

	held, ok := dbResource.Lookup(xaID.String())
	require.True(t, ok)
	assert.Same(t, heldConn, held)
	assert.Equal(t, 0, targetConn.closeCalls)
}
