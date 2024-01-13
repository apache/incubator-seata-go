package parser

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type JSONStateMachineParser struct {
	BaseStateParser
}

func NewJSONStateMachineParser() *JSONStateMachineParser {
	return &JSONStateMachineParser{
		BaseStateParser{},
	}
}

func (stateMachineParser JSONStateMachineParser) GetType() string {
	return "JSON"
}

func (stateMachineParser JSONStateMachineParser) Parse(content string) (statelang.StateMachine, error) {
	var stateMachineJsonObject StateMachineJsonObject

	err := json.Unmarshal([]byte(content), &stateMachineJsonObject)
	if err != nil {
		return nil, err
	}

	stateMachine := statelang.NewStateMachineImpl()
	stateMachine.SetName(stateMachineJsonObject.Name)
	stateMachine.SetComment(stateMachineJsonObject.Comment)
	stateMachine.SetVersion(stateMachineJsonObject.Version)
	stateMachine.SetStartState(stateMachineJsonObject.StartState)
	stateMachine.SetPersist(stateMachineJsonObject.Persist)

	if stateMachineJsonObject.Type != "" {
		stateMachine.SetType(stateMachineJsonObject.Type)
	}

	if stateMachineJsonObject.RecoverStrategy != "" {
		recoverStrategy, ok := statelang.ValueOfRecoverStrategy(stateMachineJsonObject.RecoverStrategy)
		if !ok {
			return nil, errors.New("Not support " + stateMachineJsonObject.RecoverStrategy)
		}
		stateMachine.SetRecoverStrategy(recoverStrategy)
	}

	stateParserFactory := NewDefaultStateParserFactory()
	stateParserFactory.InitDefaultStateParser()
	for stateName, v := range stateMachineJsonObject.States {
		stateMap, ok := v.(map[string]interface{})
		if !ok {
			return nil, errors.New("State [" + stateName + "] scheme illegal, required map")
		}

		stateType, ok := stateMap["Type"].(string)
		if !ok {
			return nil, errors.New("State [" + stateName + "] Type illegal, required string")
		}

		//stateMap
		stateParser := stateParserFactory.GetStateParser(stateType)
		if stateParser == nil {
			return nil, errors.New("State Type [" + stateType + "] is not support")
		}

		_, stateExist := stateMachine.States()[stateName]
		if stateExist {
			return nil, errors.New("State [name:" + stateName + "] already exists")
		}

		state, err := stateParser.Parse(stateName, stateMap)
		if err != nil {
			return nil, err
		}

		state.SetStateMachine(stateMachine)
		stateMachine.States()[stateName] = state
	}

	for _, stateValue := range stateMachine.States() {
		if stateMachineParser.isTaskState(stateValue.Type()) {
			stateMachineParser.setForCompensation(stateValue, stateMachine)
		}
	}

	return stateMachine, nil
}

func (stateMachineParser JSONStateMachineParser) setForCompensation(stateValue statelang.State, stateMachine *statelang.StateMachineImpl) {
	switch stateValue.Type() {
	case stateValue.Type():
		serviceTaskStateImpl, ok := stateValue.(*state.ServiceTaskStateImpl)
		if ok {
			if serviceTaskStateImpl.CompensateState() != "" {
				compState := stateMachine.States()[serviceTaskStateImpl.CompensateState()]
				if stateMachineParser.isTaskState(compState.Type()) {
					compStateImpl, ok := compState.(state.ServiceTaskStateImpl)
					if ok {
						compStateImpl.SetForCompensation(true)
					}
				}
			}
		}
	}
}

func (stateMachineParser JSONStateMachineParser) isTaskState(stateType string) bool {
	if stateType == constant.StateTypeServiceTask {
		return true
	}
	return false
}

type StateMachineJsonObject struct {
	Name                        string                 `json:"Name"`
	Comment                     string                 `json:"Comment"`
	Version                     string                 `json:"Version"`
	StartState                  string                 `json:"StartState"`
	RecoverStrategy             string                 `json:"RecoverStrategy"`
	Persist                     bool                   `json:"IsPersist"`
	RetryPersistModeUpdate      bool                   `json:"IsRetryPersistModeUpdate"`
	CompensatePersistModeUpdate bool                   `json:"IsCompensatePersistModeUpdate"`
	Type                        string                 `json:"Type"`
	States                      map[string]interface{} `json:"States"`
}
