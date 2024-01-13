package parser

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type CompensationTriggerStateParser struct {
	BaseStateParser
}

func NewCompensationTriggerStateParser() *CompensationTriggerStateParser {
	return &CompensationTriggerStateParser{
		BaseStateParser{},
	}
}

func (c CompensationTriggerStateParser) StateType() string {
	return constant.StateTypeCompensationTrigger
}

func (c CompensationTriggerStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	compensateSubStateMachineStateImpl := state.NewCompensationTriggerStateImpl()
	err := c.ParseBaseAttributes(stateName, compensateSubStateMachineStateImpl, stateMap)
	if err != nil {
		return nil, err
	}

	return compensateSubStateMachineStateImpl, nil
}
