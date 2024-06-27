package core

import "context"

type EventPublisher interface {
	PushEvent(ctx context.Context, event Event) (bool, error)
}

type ProcessCtrlEventPublisher struct {
	eventBus EventBus
}

func NewProcessCtrlEventPublisher(eventBus EventBus) *ProcessCtrlEventPublisher {
	return &ProcessCtrlEventPublisher{eventBus: eventBus}
}

func (p ProcessCtrlEventPublisher) PushEvent(ctx context.Context, event Event) (bool, error) {
	return p.eventBus.Offer(ctx, event)
}
