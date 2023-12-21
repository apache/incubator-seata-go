package engine

type Event interface {
}

type EventPublisher interface {
	pushEvent(event Event) (bool, error)
}
