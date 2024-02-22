package parser

import (
	"seata.apache.org/seata-go/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang/state"
)

type SucceedEndStateParser struct {
	*BaseStateParser
}

func NewSucceedEndStateParser() *SucceedEndStateParser {
	return &SucceedEndStateParser{
		&BaseStateParser{},
	}
}

func (s SucceedEndStateParser) StateType() string {
	return constant.StateTypeSucceed
}

func (s SucceedEndStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	succeedEndStateImpl := state.NewSucceedEndStateImpl()
	err := s.ParseBaseAttributes(stateName, succeedEndStateImpl, stateMap)
	if err != nil {
		return nil, err
	}

	return succeedEndStateImpl, nil
}

type FailEndStateParser struct {
	*BaseStateParser
}

func NewFailEndStateParser() *FailEndStateParser {
	return &FailEndStateParser{
		&BaseStateParser{},
	}
}

func (f FailEndStateParser) StateType() string {
	return constant.StateTypeFail
}

func (f FailEndStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	failEndStateImpl := state.NewFailEndStateImpl()
	err := f.ParseBaseAttributes(stateName, failEndStateImpl, stateMap)
	if err != nil {
		return nil, err
	}

	errorCode, err := f.GetStringOrDefault(stateName, stateMap, "ErrorCode", "")
	if err != nil {
		return nil, err
	}
	failEndStateImpl.SetErrorCode(errorCode)

	message, err := f.GetStringOrDefault(stateName, stateMap, "Message", "")
	if err != nil {
		return nil, err
	}
	failEndStateImpl.SetMessage(message)
	return failEndStateImpl, nil
}
