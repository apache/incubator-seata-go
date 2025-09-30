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

package store

import (
	"context"
	"seata-go-ai-workflow-agent/pkg/session/config"
	"seata-go-ai-workflow-agent/pkg/session/types"
	"sort"
	"sync"
	"time"
)

// MemoryStore implements SessionStore using in-memory storage
type MemoryStore struct {
	mu          sync.RWMutex
	sessions    map[string]*types.SessionData
	userIndex   map[string][]string
	accessTimes map[string]time.Time
	config      config.MemoryConfig
	cleanup     *time.Ticker
	stats       types.StoreStats
	startTime   time.Time
	closed      chan struct{}
}

// NewMemoryStore creates a new memory-based session store
func NewMemoryStore(cfg config.MemoryConfig) *MemoryStore {
	store := &MemoryStore{
		sessions:    make(map[string]*types.SessionData),
		userIndex:   make(map[string][]string),
		accessTimes: make(map[string]time.Time),
		config:      cfg,
		startTime:   time.Now(),
		closed:      make(chan struct{}),
	}

	if cfg.CleanupInterval > 0 {
		store.cleanup = time.NewTicker(cfg.CleanupInterval)
		go store.cleanupWorker()
	}

	return store
}

// Get retrieves a session by ID
func (m *MemoryStore) Get(ctx context.Context, sessionID string) (*types.SessionData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, types.ErrSessionNotFound
	}

	if session.IsExpired() {
		return nil, types.ErrSessionExpired
	}

	// Update access time for LRU
	m.accessTimes[sessionID] = time.Now()

	return session.Clone(), nil
}

// Set stores or updates a session
func (m *MemoryStore) Set(ctx context.Context, session *types.SessionData) error {
	if session == nil {
		return types.ErrInvalidSession
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check capacity and evict if necessary
	if err := m.ensureCapacity(); err != nil {
		return err
	}

	// Clone to avoid external mutations
	sessionCopy := session.Clone()
	sessionCopy.UpdatedAt = time.Now()

	// Update main storage
	m.sessions[session.ID] = sessionCopy
	m.accessTimes[session.ID] = time.Now()

	// Update user index
	m.updateUserIndex(session.UserID, session.ID)

	// Update stats
	m.updateStats()

	return nil
}

// Delete removes a session
func (m *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return types.ErrSessionNotFound
	}

	// Remove from main storage
	delete(m.sessions, sessionID)
	delete(m.accessTimes, sessionID)

	// Remove from user index
	m.removeFromUserIndex(session.UserID, sessionID)

	// Update stats
	m.updateStats()

	return nil
}

// Exists checks if a session exists
func (m *MemoryStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false, nil
	}

	if session.IsExpired() {
		return false, nil
	}

	return true, nil
}

// List returns session IDs for a user with pagination
func (m *MemoryStore) List(ctx context.Context, userID string, offset, limit int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionIDs, exists := m.userIndex[userID]
	if !exists {
		return []string{}, nil
	}

	// Filter expired sessions
	var validIDs []string
	for _, id := range sessionIDs {
		if session, exists := m.sessions[id]; exists && !session.IsExpired() {
			validIDs = append(validIDs, id)
		}
	}

	// Sort by update time (newest first)
	sort.Slice(validIDs, func(i, j int) bool {
		sessionI := m.sessions[validIDs[i]]
		sessionJ := m.sessions[validIDs[j]]
		return sessionI.UpdatedAt.After(sessionJ.UpdatedAt)
	})

	// Apply pagination
	end := offset + limit
	if offset > len(validIDs) {
		return []string{}, nil
	}
	if end > len(validIDs) {
		end = len(validIDs)
	}

	return validIDs[offset:end], nil
}

// Count returns the total number of sessions for a user
func (m *MemoryStore) Count(ctx context.Context, userID string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionIDs, exists := m.userIndex[userID]
	if !exists {
		return 0, nil
	}

	count := int64(0)
	for _, id := range sessionIDs {
		if session, exists := m.sessions[id]; exists && !session.IsExpired() {
			count++
		}
	}

	return count, nil
}

// Cleanup removes expired sessions
func (m *MemoryStore) Cleanup(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := int64(0)
	now := time.Now()

	for id, session := range m.sessions {
		if session.IsExpired() {
			delete(m.sessions, id)
			delete(m.accessTimes, id)
			m.removeFromUserIndex(session.UserID, id)
			removed++
		}
	}

	m.stats.LastCleanup = &now
	m.updateStats()

	return removed, nil
}

// Close closes the store and releases resources
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cleanup != nil {
		m.cleanup.Stop()
	}

	close(m.closed)
	return nil
}

// Health checks store connectivity
func (m *MemoryStore) Health(ctx context.Context) error {
	select {
	case <-m.closed:
		return types.ErrStoreUnavailable
	default:
		return nil
	}
}

// GetStats returns store statistics
func (m *MemoryStore) GetStats() types.StoreStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := m.stats
	stats.TotalSessions = int64(len(m.sessions))

	var oldestTime, newestTime *time.Time
	var totalMessages int64

	for _, session := range m.sessions {
		if !session.IsExpired() {
			stats.ActiveSessions++
			totalMessages += int64(session.MessageCount)

			if oldestTime == nil || session.CreatedAt.Before(*oldestTime) {
				oldestTime = &session.CreatedAt
			}
			if newestTime == nil || session.CreatedAt.After(*newestTime) {
				newestTime = &session.CreatedAt
			}
		} else {
			stats.ExpiredSessions++
		}
	}

	if stats.ActiveSessions > 0 {
		stats.AverageMessages = float64(totalMessages) / float64(stats.ActiveSessions)
	}

	stats.OldestSession = oldestTime
	stats.NewestSession = newestTime

	return stats
}

// ensureCapacity checks capacity and evicts sessions if necessary
func (m *MemoryStore) ensureCapacity() error {
	if m.config.MaxSessions <= 0 || len(m.sessions) < m.config.MaxSessions {
		return nil
	}

	// Evict based on policy
	switch m.config.EvictionPolicy {
	case "lru":
		return m.evictLRU()
	case "lfu":
		return m.evictLFU()
	default:
		return m.evictOldest()
	}
}

// evictLRU evicts least recently used session
func (m *MemoryStore) evictLRU() error {
	var oldestID string
	var oldestTime time.Time

	for id, accessTime := range m.accessTimes {
		if oldestID == "" || accessTime.Before(oldestTime) {
			oldestID = id
			oldestTime = accessTime
		}
	}

	if oldestID != "" {
		session := m.sessions[oldestID]
		delete(m.sessions, oldestID)
		delete(m.accessTimes, oldestID)
		if session != nil {
			m.removeFromUserIndex(session.UserID, oldestID)
		}
	}

	return nil
}

// evictLFU evicts least frequently used session (simplified as oldest access)
func (m *MemoryStore) evictLFU() error {
	return m.evictLRU() // Simplified implementation
}

// evictOldest evicts the oldest session
func (m *MemoryStore) evictOldest() error {
	var oldestID string
	var oldestTime time.Time

	for id, session := range m.sessions {
		if oldestID == "" || session.CreatedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = session.CreatedAt
		}
	}

	if oldestID != "" {
		session := m.sessions[oldestID]
		delete(m.sessions, oldestID)
		delete(m.accessTimes, oldestID)
		if session != nil {
			m.removeFromUserIndex(session.UserID, oldestID)
		}
	}

	return nil
}

// updateUserIndex updates the user session index
func (m *MemoryStore) updateUserIndex(userID, sessionID string) {
	sessionIDs := m.userIndex[userID]

	// Check if already exists
	for _, id := range sessionIDs {
		if id == sessionID {
			return
		}
	}

	m.userIndex[userID] = append(sessionIDs, sessionID)
}

// removeFromUserIndex removes a session from user index
func (m *MemoryStore) removeFromUserIndex(userID, sessionID string) {
	sessionIDs := m.userIndex[userID]
	for i, id := range sessionIDs {
		if id == sessionID {
			m.userIndex[userID] = append(sessionIDs[:i], sessionIDs[i+1:]...)
			break
		}
	}

	if len(m.userIndex[userID]) == 0 {
		delete(m.userIndex, userID)
	}
}

// updateStats updates internal statistics
func (m *MemoryStore) updateStats() {
	// Basic stats are updated in GetStats() for thread safety
}

// cleanupWorker runs periodic cleanup
func (m *MemoryStore) cleanupWorker() {
	for {
		select {
		case <-m.cleanup.C:
			m.Cleanup(context.Background())
		case <-m.closed:
			return
		}
	}
}
