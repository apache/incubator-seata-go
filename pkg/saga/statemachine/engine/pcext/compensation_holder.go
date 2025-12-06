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
	"sync"

	"seata.apache.org/seata-go/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/pkg/util/collection"
)

type CompensationHolder struct {
	statesNeedCompensation     *sync.Map
	statesForCompensation      *sync.Map
	stateStackNeedCompensation *collection.Stack
}

func (c *CompensationHolder) StatesNeedCompensation() *sync.Map {
	return c.statesNeedCompensation
}

func (c *CompensationHolder) SetStatesNeedCompensation(statesNeedCompensation *sync.Map) {
	c.statesNeedCompensation = statesNeedCompensation
}

func (c *CompensationHolder) StatesForCompensation() *sync.Map {
	return c.statesForCompensation
}

func (c *CompensationHolder) SetStatesForCompensation(statesForCompensation *sync.Map) {
	c.statesForCompensation = statesForCompensation
}

func (c *CompensationHolder) StateStackNeedCompensation() *collection.Stack {
	return c.stateStackNeedCompensation
}

func (c *CompensationHolder) SetStateStackNeedCompensation(stateStackNeedCompensation *collection.Stack) {
	c.stateStackNeedCompensation = stateStackNeedCompensation
}

func (c *CompensationHolder) AddToBeCompensatedState(stateName string, toBeCompensatedState statelang.StateInstance) {
	c.statesNeedCompensation.Store(stateName, toBeCompensatedState)
}

func NewCompensationHolder() *CompensationHolder {
	return &CompensationHolder{
		statesNeedCompensation:     &sync.Map{},
		statesForCompensation:      &sync.Map{},
		stateStackNeedCompensation: collection.NewStack(),
	}
}

func GetCurrentCompensationHolder(ctx context.Context, processContext process_ctrl.ProcessContext, forceCreate bool) *CompensationHolder {
	var holder *CompensationHolder
	if v := processContext.GetVariable(constant.VarNameCurrentCompensationHolder); v != nil {
		if h, ok := v.(*CompensationHolder); ok {
			holder = h
		}
	}
	// ensure context mutex exists
	lock, _ := processContext.GetVariable(constant.VarNameProcessContextMutexLock).(*sync.Mutex)
	if lock == nil {
		lock = &sync.Mutex{}
		processContext.SetVariable(constant.VarNameProcessContextMutexLock, lock)
	}
	lock.Lock()
	defer lock.Unlock()
	if holder == nil && forceCreate {
		holder = NewCompensationHolder()
		processContext.SetVariable(constant.VarNameCurrentCompensationHolder, holder)
	}
	return holder
}
