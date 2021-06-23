package registry

// EventType means SourceObjectEventType
type EventType int

// ServiceEvent includes create, update, delete event
type ServiceEvent struct {
	Action EventType
	// store the key for Service.Key()
	key string
	// If the url is updated, such as Merged.
	updated bool
}
