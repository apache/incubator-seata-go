package events

import (
	"context"
)

type EventBus interface {
	Offer(ctx context.Context, event Event) (bool, error)

	RegisterEventConsumer(consumer EventConsumer)
}

type BaseEventBus struct {
	eventConsumerList []EventConsumer
}

func (b *BaseEventBus) RegisterEventConsumer(consumer EventConsumer) {
	if b.eventConsumerList == nil {
		b.eventConsumerList = make([]EventConsumer, 0)
	}
	b.eventConsumerList = append(b.eventConsumerList, consumer)
}

func (b *BaseEventBus) GetEventConsumerList(event Event) []EventConsumer {
	var acceptedConsumerList = make([]EventConsumer, 0)
	for i := range b.eventConsumerList {
		eventConsumer := b.eventConsumerList[i]
		if eventConsumer.Accept(event) {
			acceptedConsumerList = append(acceptedConsumerList, eventConsumer)
		}
	}
	return acceptedConsumerList
}

type DirectEventBus struct {
	BaseEventBus
}

func (d DirectEventBus) Offer(ctx context.Context, event Event) (bool, error) {
	eventConsumerList := d.GetEventConsumerList(event)
	if len(eventConsumerList) == 0 {
		//TODO logger
		return false, nil
	}

	//processContext, _ := event.(process_ctrl.ProcessContext)

	return true, nil
}
