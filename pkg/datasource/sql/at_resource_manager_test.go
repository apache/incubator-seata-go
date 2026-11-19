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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	serr "seata.apache.org/seata-go/v2/pkg/util/errors"
)

type registrationRemoting struct {
	rm.RMRemoting
	register func(rm.Resource) error
}

func (r *registrationRemoting) RegisterResource(resource rm.Resource) error {
	return r.register(resource)
}

func TestRegisterATResourceDuringReconnect(t *testing.T) {
	retryErr := errors.New("replacement session unavailable")
	for _, tt := range []struct {
		name            string
		reconnects      int
		wantSessions    []int
		rpcErr          error
		wantUnstableErr bool
	}{
		{name: "register on the new session", reconnects: 1, wantSessions: []int{1, 2}},
		{name: "keep resource unpublished when retry fails", reconnects: 1, wantSessions: []int{1, 2}, rpcErr: retryErr},
		{name: "allow more than one reconnect", reconnects: 2, wantSessions: []int{1, 2, 3}},
		{name: "stop retrying when reconnects continue", reconnects: 3, wantSessions: []int{1, 2, 3}, wantUnstableErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ATSourceManager{}
			cache := &rm.ResourceManagerCache{}
			cache.RegisterResourceManager(manager)
			resource := &DBResource{resourceID: "db"}
			session := 1
			var registeredSessions []int
			manager.rmRemoting = &registrationRemoting{register: func(res rm.Resource) error {
				require.Same(t, resource, res)
				_, published := manager.resourceCache.Load(resource.resourceID)
				require.False(t, published)
				registeredSessions = append(registeredSessions, session)
				if session <= tt.reconnects {
					// The old session's reply reaches the caller after the new session replays.
					session++
					require.NoError(t, cache.RegisterCachedResources())
					return nil
				}
				return tt.rpcErr
			}}

			err := manager.RegisterResource(resource)
			if tt.wantUnstableErr {
				require.ErrorContains(t, err, "AT resource registration interrupted by reconnect")
			} else {
				require.ErrorIs(t, err, tt.rpcErr)
			}
			require.Equal(t, tt.wantSessions, registeredSessions)
			cached, published := manager.resourceCache.Load(resource.resourceID)
			require.Equal(t, tt.rpcErr == nil && !tt.wantUnstableErr, published)
			if published {
				require.Same(t, resource, cached)
			}
		})
	}
}

func TestRegisterCachedATResourcesDoesNotOverwriteReplacement(t *testing.T) {
	original := &DBResource{resourceID: "db"}
	replacement := &DBResource{resourceID: original.resourceID}
	replayStarted, releaseReplay := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseReplay) }) }
	t.Cleanup(release)
	manager := &ATSourceManager{rmRemoting: &registrationRemoting{register: func(resource rm.Resource) error {
		if resource == original {
			close(replayStarted)
			<-releaseReplay
		}
		return nil
	}}}
	manager.resourceCache.Store(original.resourceID, original)
	cache := &rm.ResourceManagerCache{}
	cache.RegisterResourceManager(manager)

	replayResult := make(chan error, 1)
	go func() { replayResult <- cache.RegisterCachedResources() }()
	select {
	case <-replayStarted:
	case <-time.After(time.Second):
		t.Fatal("reconnect did not register the cached resource")
	}
	registerErr := manager.RegisterResource(replacement)
	release()
	require.NoError(t, registerErr)
	require.NoError(t, <-replayResult)
	cached, ok := manager.resourceCache.Load(replacement.resourceID)
	require.True(t, ok)
	require.Same(t, replacement, cached)
}

type rollbackErrorUndoManager struct {
	undo.UndoLogManager
	err error
}

func (m *rollbackErrorUndoManager) RunUndo(context.Context, string, int64, *sql.DB, string, types.TableMetaReader) error {
	return m.err
}

func TestBranchRollbackPreservesFailureStatus(t *testing.T) {
	undoErr := errors.New("undo failed")
	for _, tt := range []struct {
		name       string
		resource   any
		wantStatus branch.BranchStatus
		wantErr    string
	}{
		{"missing resource", nil, branch.BranchStatusUnknown, "resourceId: db"},
		{"invalid resource", "invalid", branch.BranchStatusUnknown, "invalid AT resource"},
		{"missing manager", &DBResource{}, branch.BranchStatusUnknown, "no Undo Log Manager"},
		{"plain error", &DBResource{undoLogMgr: &rollbackErrorUndoManager{err: undoErr}}, branch.BranchStatusPhaseoneFailed, "undo failed"},
		{"retryable", &DBResource{undoLogMgr: &rollbackErrorUndoManager{err: serr.New(serr.TransactionErrorCodeBranchRollbackFailedRetriable, "", undoErr)}}, branch.BranchStatusPhasetwoRollbackFailedRetryable, ""},
		{"nonretryable", &DBResource{undoLogMgr: &rollbackErrorUndoManager{err: serr.New(serr.TransactionErrorCodeBranchRollbackFailedUnretriable, "", undoErr)}}, branch.BranchStatusPhasetwoRollbackFailedUnretryable, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ATSourceManager{}
			if tt.resource != nil {
				manager.resourceCache.Store("db", tt.resource)
			}
			status, err := manager.BranchRollback(context.Background(), rm.BranchResource{ResourceId: "db"})
			require.Equal(t, tt.wantStatus, status)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
