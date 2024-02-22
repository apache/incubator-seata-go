package events

import (
	"context"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/process_ctrl"
)

type EventConsumer interface {
	Accept(event Event) bool

	Process(ctx context.Context, event Event) error
}

type ProcessCtrlEventConsumer struct {
}

func (p ProcessCtrlEventConsumer) Accept(event Event) bool {
	if event == nil {
		return false
	}

	_, ok := event.(process_ctrl.ProcessContext)
	return ok
}

func (p ProcessCtrlEventConsumer) Process(ctx context.Context, event Event) error {
	//TODO implement me
	panic("implement me")
}
