package parser

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type ChoiceStateParser struct {
	BaseStateParser
}

func NewChoiceStateParser() *ChoiceStateParser {
	return &ChoiceStateParser{}
}

func (c ChoiceStateParser) StateType() string {
	return "Choice"
}

func (c ChoiceStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	choiceState := state.NewChoiceStateImpl()
	err := c.ParseBaseAttributes(stateName, choiceState, stateMap)
	if err != nil {
		return nil, err
	}

	//TODO choice parse
	return choiceState, nil
}
