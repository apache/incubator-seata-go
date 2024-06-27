package core

import (
	"context"
	"seata.apache.org/seata-go/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/pkg/util/collection"
	"sync"
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

func GetCurrentCompensationHolder(ctx context.Context, processContext ProcessContext, forceCreate bool) *CompensationHolder {
	compensationholder := processContext.GetVariable(constant.VarNameCurrentCompensationHolder).(*CompensationHolder)
	lock := processContext.GetVariable(constant.VarNameProcessContextMutexLock).(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if compensationholder == nil && forceCreate {
		compensationholder = NewCompensationHolder()
		processContext.SetVariable(constant.VarNameCurrentCompensationHolder, compensationholder)
	}
	return compensationholder
}
