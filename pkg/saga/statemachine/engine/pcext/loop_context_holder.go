/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pcext

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
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

func GetCurrentLoopContextHolder(ctx context.Context, processContext process_ctrl.ProcessContext, forceCreate bool) *LoopContextHolder {
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

func ClearCurrent(ctx context.Context, processContext process_ctrl.ProcessContext) {
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
