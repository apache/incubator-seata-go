package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agenthub/pkg/common"
	"agenthub/pkg/utils"
)

// MemoryStorage implements in-memory storage following K8s patterns
type MemoryStorage struct {
	resources map[string]Resource
	mutex     sync.RWMutex
	watchers  []chan Event
	logger    *utils.Logger
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		resources: make(map[string]Resource),
		watchers:  make([]chan Event, 0),
		logger:    utils.WithField("component", "memory-storage"),
	}
}

// Create creates a new resource
func (m *MemoryStorage) Create(ctx context.Context, resource Resource) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	id := resource.GetID()
	if id == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	
	if _, exists := m.resources[id]; exists {
		return fmt.Errorf("resource with ID %s already exists", id)
	}
	
	// Update timestamps if it's a BaseResource
	if baseResource, ok := resource.(*common.BaseResource); ok {
		now := time.Now()
		baseResource.CreatedAt = now
		baseResource.UpdatedAt = now
	}
	
	m.resources[id] = resource
	m.logger.Debug("Created resource %s of kind %s", id, resource.GetKind())
	
	// Notify watchers
	event := Event{
		Type:     EventTypeCreate,
		Resource: resource,
		Time:     time.Now(),
	}
	m.notifyWatchers(event)
	
	return nil
}

// Get retrieves a resource by ID
func (m *MemoryStorage) Get(ctx context.Context, id string) (Resource, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	resource, exists := m.resources[id]
	if !exists {
		return nil, fmt.Errorf("resource with ID %s not found", id)
	}
	
	m.logger.Debug("Retrieved resource %s", id)
	return resource, nil
}

// Update updates an existing resource
func (m *MemoryStorage) Update(ctx context.Context, resource Resource) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	id := resource.GetID()
	if id == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	
	if _, exists := m.resources[id]; !exists {
		return fmt.Errorf("resource with ID %s not found", id)
	}
	
	// Update timestamp if it's a BaseResource
	if baseResource, ok := resource.(*common.BaseResource); ok {
		baseResource.UpdatedAt = time.Now()
	}
	
	m.resources[id] = resource
	m.logger.Debug("Updated resource %s of kind %s", id, resource.GetKind())
	
	// Notify watchers
	event := Event{
		Type:     EventTypeUpdate,
		Resource: resource,
		Time:     time.Now(),
	}
	m.notifyWatchers(event)
	
	return nil
}

// Delete deletes a resource by ID
func (m *MemoryStorage) Delete(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	resource, exists := m.resources[id]
	if !exists {
		return fmt.Errorf("resource with ID %s not found", id)
	}
	
	delete(m.resources, id)
	m.logger.Debug("Deleted resource %s of kind %s", id, resource.GetKind())
	
	// Notify watchers
	event := Event{
		Type:     EventTypeDelete,
		Resource: resource,
		Time:     time.Now(),
	}
	m.notifyWatchers(event)
	
	return nil
}

// List lists resources with optional filters
func (m *MemoryStorage) List(ctx context.Context, filters map[string]interface{}) ([]Resource, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var resources []Resource
	for _, resource := range m.resources {
		if m.matchesFilters(resource, filters) {
			resources = append(resources, resource)
		}
	}
	
	m.logger.Debug("Listed %d resources with filters: %v", len(resources), filters)
	return resources, nil
}

// Watch watches for changes to resources
func (m *MemoryStorage) Watch(ctx context.Context, filters map[string]interface{}) (<-chan Event, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	eventChan := make(chan Event, 100) // Buffer events
	m.watchers = append(m.watchers, eventChan)
	
	m.logger.Debug("Created watcher with filters: %v", filters)
	
	// Start a goroutine to handle context cancellation
	go func() {
		<-ctx.Done()
		m.removeWatcher(eventChan)
		close(eventChan)
	}()
	
	return eventChan, nil
}

// Close closes the storage connection
func (m *MemoryStorage) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Close all watchers
	for _, watcher := range m.watchers {
		close(watcher)
	}
	m.watchers = nil
	
	m.logger.Info("Memory storage closed")
	return nil
}

// HealthCheck checks storage health
func (m *MemoryStorage) HealthCheck(ctx context.Context) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Simple health check - ensure we can access resources map
	_ = len(m.resources)
	return nil
}

// matchesFilters checks if a resource matches the given filters
func (m *MemoryStorage) matchesFilters(resource Resource, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}
	
	for key, value := range filters {
		switch key {
		case "kind":
			if resource.GetKind() != value {
				return false
			}
		case "version":
			if resource.GetVersion() != value {
				return false
			}
		default:
			// Check metadata
			metadata := resource.GetMetadata()
			if metadataValue, exists := metadata[key]; !exists || metadataValue != value {
				return false
			}
		}
	}
	
	return true
}

// notifyWatchers notifies all watchers of an event
func (m *MemoryStorage) notifyWatchers(event Event) {
	for _, watcher := range m.watchers {
		select {
		case watcher <- event:
		default:
			// Channel is full, skip this watcher
			m.logger.Warn("Watcher channel full, dropping event")
		}
	}
}

// removeWatcher removes a watcher from the list
func (m *MemoryStorage) removeWatcher(watcher chan Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for i, w := range m.watchers {
		if w == watcher {
			m.watchers = append(m.watchers[:i], m.watchers[i+1:]...)
			break
		}
	}
}