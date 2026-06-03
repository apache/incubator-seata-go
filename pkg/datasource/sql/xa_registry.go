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
	"sync"
	"time"

	"seata.apache.org/seata-go/v2/pkg/util/log"
)

// xaState represents the state of an XA transaction branch
type xaState int

const (
	xaStateIdle     xaState = iota // No XA transaction
	xaStateStarted                  // XA START executed
	xaStateEnded                    // XA END executed
	xaStatePrepared                 // XA PREPARE executed
)

// xaEntry represents an active XA transaction branch
type xaEntry struct {
	conn           *XAConn
	xid            string
	branchID       string
	resourceID     string
	state          xaState
	createTime     time.Time
	lastAccessTime time.Time
	statementCount int
}

// xaRegistry manages XA connections for global transactions
// It ensures multiple SQL operations in the same global transaction
// reuse the same XA branch, preventing "busy buffer" errors
type xaRegistry struct {
	mu      sync.RWMutex
	entries map[string]*xaEntry // key: xid
}

var (
	globalRegistry *xaRegistry
	registryOnce   sync.Once
)

// getXARegistry returns the global XA registry (singleton)
func getXARegistry() *xaRegistry {
	registryOnce.Do(func() {
		globalRegistry = &xaRegistry{
			entries: make(map[string]*xaEntry),
		}
	})
	return globalRegistry
}

// register registers or retrieves an XA connection for the given xid
// Returns (isNew, entry, error)
func (r *xaRegistry) register(xid, branchID, resourceID string, conn *XAConn) (bool, *xaEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Check if entry already exists
	if entry, ok := r.entries[xid]; ok {
		entry.lastAccessTime = now
		entry.statementCount++
		log.Infof("Reusing existing XA branch for xid: %s, branchID: %s, statementCount: %d, state: %v",
			xid, entry.branchID, entry.statementCount, entry.state)
		return false, entry
	}

	// Create new entry
	entry := &xaEntry{
		conn:           conn,
		xid:            xid,
		branchID:       branchID,
		resourceID:     resourceID,
		state:          xaStateStarted,
		createTime:     now,
		lastAccessTime: now,
		statementCount: 1,
	}

	r.entries[xid] = entry
	log.Infof("Registered new XA branch, xid: %s, branchID: %s, resourceID: %s", xid, branchID, resourceID)

	return true, entry
}

// get retrieves an XA entry by xid
func (r *xaRegistry) get(xid string) (*xaEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[xid]
	if ok {
		entry.lastAccessTime = time.Now()
	}
	return entry, ok
}

// canReuse checks if an XA connection can be reused for the given xid
func (r *xaRegistry) canReuse(xid string) bool {
	entry, ok := r.get(xid)
	if !ok {
		return false
	}

	// Can reuse if: entry exists, connection is active, and not yet ended/prepared
	// Also check that the connection is the same (for connection pooling)
	if entry.conn == nil {
		return false
	}
	return entry.state == xaStateStarted && entry.conn.xaActive && entry.branchID != "pending"
}

// setState updates the state of an XA entry
func (r *xaRegistry) setState(xid string, state xaState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.entries[xid]; ok {
		oldState := entry.state
		entry.state = state
		log.Infof("XA state changed, xid: %s, branchID: %s, %v -> %v",
			xid, entry.branchID, oldState, state)
	}
}

// unregister removes an XA entry from the registry
func (r *xaRegistry) unregister(xid string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.entries[xid]; ok {
		log.Infof("Unregistered XA branch, xid: %s, branchID: %s, statementCount: %d",
			xid, entry.branchID, entry.statementCount)
		delete(r.entries, xid)
	}
}

// cleanup removes stale entries (called periodically)
func (r *xaRegistry) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	maxLifetime := 30 * time.Minute

	for xid, entry := range r.entries {
		// Remove entries that are too old or have inactive connections
		// Check for nil conn before accessing xaActive
		stale := now.Sub(entry.createTime) > maxLifetime
		inactive := entry.conn != nil && !entry.conn.xaActive
		if stale || inactive {
			log.Infof("Cleaning up stale XA entry, xid: %s, branchID: %s", xid, entry.branchID)
			delete(r.entries, xid)
		}
	}
}

// getActiveCount returns the number of active XA entries
func (r *xaRegistry) getActiveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, entry := range r.entries {
		// Check conn is not nil before accessing xaActive
		if entry.state == xaStateStarted && entry.conn != nil && entry.conn.xaActive {
			count++
		}
	}
	return count
}
