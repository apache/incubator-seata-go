package parser

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
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
	return engine.StateTypeChoice
}

func (c ChoiceStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	choiceState := state.NewChoiceStateImpl()
	choiceState.SetName(stateName)

	//parse Type
	typeName, err := c.GetString(stateName, stateMap, "Type")
	if err != nil {
		return nil, err
	}
	choiceState.SetType(typeName)

	//parse Default
	defaultChoice, err := c.GetString(stateName, stateMap, "Default")
	if err != nil {
		return nil, err
	}
	choiceState.SetDefault(defaultChoice)

	//parse Choices
	slice, err := c.GetSlice(stateName, stateMap, "Choices")
	if err != nil {
		return nil, err
	}

	var choices []state.Choice
	for i := range slice {
		choiceValMap, ok := slice[i].(map[string]interface{})
		if !ok {
			return nil, errors.New(fmt.Sprintf("State [%s] Choices element required struct", stateName))
		}

		choice := state.NewChoiceImpl()
		expression, err := c.GetString(stateName, choiceValMap, "Expression")
		if err != nil {
			return nil, err
		}
		choice.SetExpression(expression)

		next, err := c.GetString(stateName, choiceValMap, "Next")
		if err != nil {
			return nil, err
		}
		choice.SetNext(next)

		choices = append(choices, choice)
	}
	choiceState.SetChoices(choices)

	return choiceState, nil
}
