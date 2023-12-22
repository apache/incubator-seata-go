package engine

import (
	"context"
)

type Event interface {
}

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

type EventConsumer interface {
	Accept(event Event) bool

	Process(ctx context.Context, event Event) error
}

type ProcessCtrlEventConsumer struct {
}

func (p ProcessCtrlEventConsumer) Accept(event Event) bool {
	_, ok := event.(ProcessContext)
	return ok
}

func (p ProcessCtrlEventConsumer) Process(ctx context.Context, event Event) error {
	//TODO implement me
	panic("implement me")
}

type EventBus interface {
	Offer(ctx context.Context, event Event) (bool, error)

	RegisterEventConsumer(consumer EventConsumer)
}
