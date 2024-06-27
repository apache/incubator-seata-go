package core

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"sync"
)

type LoopContextHolder struct {
	nrOfInstances                int32
	nrOfActiveInstances          int32
	nrOfCompletedInstances       int32
	failEnd                      bool
	completionConditionSatisfied bool
	loopCounterStack             []int
	forwardCounterStack          []int
	collection                   interface{}
}

func NewLoopContextHolder() *LoopContextHolder {
	return &LoopContextHolder{
		nrOfInstances:                0,
		nrOfActiveInstances:          0,
		nrOfCompletedInstances:       0,
		failEnd:                      false,
		completionConditionSatisfied: false,
		loopCounterStack:             make([]int, 0),
		forwardCounterStack:          make([]int, 0),
		collection:                   nil,
	}
}

func GetCurrentLoopContextHolder(ctx context.Context, processContext ProcessContext, forceCreate bool) *LoopContextHolder {
	mutex := processContext.GetVariable(constant.VarNameProcessContextMutexLock).(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	loopContextHolder := processContext.GetVariable(constant.VarNameCurrentLoopContextHolder).(*LoopContextHolder)
	if loopContextHolder == nil && forceCreate {
		loopContextHolder = &LoopContextHolder{}
		processContext.SetVariable(constant.VarNameCurrentLoopContextHolder, loopContextHolder)
	}
	return loopContextHolder
}

func ClearCurrent(ctx context.Context, processContext ProcessContext) {
	processContext.RemoveVariable(constant.VarNameCurrentLoopContextHolder)
}

func (l *LoopContextHolder) NrOfInstances() int32 {
	return l.nrOfInstances
}

func (l *LoopContextHolder) NrOfActiveInstances() int32 {
	return l.nrOfActiveInstances
}

func (l *LoopContextHolder) NrOfCompletedInstances() int32 {
	return l.nrOfCompletedInstances
}

func (l *LoopContextHolder) FailEnd() bool {
	return l.failEnd
}

func (l *LoopContextHolder) SetFailEnd(failEnd bool) {
	l.failEnd = failEnd
}

func (l *LoopContextHolder) CompletionConditionSatisfied() bool {
	return l.completionConditionSatisfied
}

func (l *LoopContextHolder) SetCompletionConditionSatisfied(completionConditionSatisfied bool) {
	l.completionConditionSatisfied = completionConditionSatisfied
}

func (l *LoopContextHolder) LoopCounterStack() []int {
	return l.loopCounterStack
}

func (l *LoopContextHolder) ForwardCounterStack() []int {
	return l.forwardCounterStack
}

func (l *LoopContextHolder) Collection() interface{} {
	return l.collection
}

func (l *LoopContextHolder) SetCollection(collection interface{}) {
	l.collection = collection
}
