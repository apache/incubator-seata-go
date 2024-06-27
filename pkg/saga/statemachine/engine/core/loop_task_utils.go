package core

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/util/log"
)

func GetLoopConfig(ctx context.Context, processContext ProcessContext, currentState statelang.State) state.Loop {
	if matchLoop(currentState) {
		taskState := currentState.(state.AbstractTaskState)
		stateMachineInstance := processContext.GetVariable(constant.VarNameStateMachineInst).(statelang.StateMachineInstance)
		stateMachineConfig := processContext.GetVariable(constant.VarNameStateMachineConfig).(StateMachineConfig)

		if taskState.Loop() != nil {
			loop := taskState.Loop()
			collectionName := loop.Collection()
			if collectionName != "" {
				expression := CreateValueExpression(stateMachineConfig.ExpressionResolver(), collectionName)
				collection := GetValue(expression, stateMachineInstance.Context(), nil)
				collectionList := collection.([]any)
				if len(collectionList) > 0 {
					current := GetCurrentLoopContextHolder(ctx, processContext, true)
					current.SetCollection(collection)
					return loop
				}
			}
			log.Warn("State [{}] loop collection param [{}] invalid", currentState.Name(), collectionName)
		}

	}
	return nil
}

func matchLoop(currentState statelang.State) bool {
	return currentState != nil && (constant.StateTypeServiceTask == currentState.Type() ||
		constant.StateTypeScriptTask == currentState.Type() || constant.StateTypeSubStateMachine == currentState.Type())
}
