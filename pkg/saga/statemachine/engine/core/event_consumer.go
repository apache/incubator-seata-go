package core

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

type EventConsumer interface {
	Accept(event Event) bool

	Process(ctx context.Context, event Event) error
}

type ProcessCtrlEventConsumer struct {
	processController ProcessController
}

func (p ProcessCtrlEventConsumer) Accept(event Event) bool {
	if event == nil {
		return false
	}

	_, ok := event.(ProcessContext)
	return ok
}

func (p ProcessCtrlEventConsumer) Process(ctx context.Context, event Event) error {
	processContext, ok := event.(ProcessContext)
	if !ok {
		return errors.New(fmt.Sprint("event %T is illegal, required process_ctrl.ProcessContext", event))
	}
	return p.processController.Process(ctx, processContext)
}
