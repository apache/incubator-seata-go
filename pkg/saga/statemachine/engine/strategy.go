package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StatusDecisionStrategy interface {
	// DecideOnEndState Determine state machine execution status when executing to EndState
	DecideOnEndState(ctx context.Context, processContext process_ctrl.ProcessContext,
		stateMachineInstance statelang.StateMachineInstance, exp error) error
	// DecideOnTaskStateFail Determine state machine execution status when executing TaskState error
	DecideOnTaskStateFail(ctx context.Context, processContext process_ctrl.ProcessContext,
		stateMachineInstance statelang.StateMachineInstance, exp error) error
	// DecideMachineForwardExecutionStatus Determine the forward execution state of the state machine
	DecideMachineForwardExecutionStatus(ctx context.Context,
		stateMachineInstance statelang.StateMachineInstance, exp error, specialPolicy bool) error
}
