package parser

import (
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"sync"
)

type StateMachineParser interface {
	GetType() string
	Parse(content string) (statelang.StateMachine, error)
}

type StateParser interface {
	StateType() string
	Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error)
}

type BaseStateParser struct {
}

func (b BaseStateParser) ParseBaseAttributes(stateName string, state statelang.State, stateMap map[string]interface{}) error {
	state.SetName(stateName)
	state.SetComment(b.GetStringOrDefault(stateMap, "Comment", ""))
	state.SetNext(b.GetStringOrDefault(stateMap, "Next", ""))
	return nil
}

func (b BaseStateParser) GetString(stateName string, stateMap map[string]interface{}, key string) (string, error) {
	value := stateMap[key]
	if value == nil {
		var result string
		return result, errors.New("State [" + stateName + "] " + key + " not exist")
	}

	valueAsString, ok := value.(string)
	if !ok {
		var s string
		return s, errors.New("State [" + stateName + "] " + key + " illegal, required string")
	}
	return valueAsString, nil
}

func (b BaseStateParser) GetStringOrDefault(stateMap map[string]interface{}, key string, defaultValue string) string {
	value := stateMap[key]
	if value == nil {
		return defaultValue
	}

	valueAsString, ok := value.(string)
	if !ok {
		return defaultValue
	}
	return valueAsString
}

func (b BaseStateParser) GetSlice(stateName string, stateMap map[string]interface{}, key string) ([]interface{}, error) {
	value := stateMap[key]
	if value == nil {
		var result []interface{}
		return result, errors.New("State [" + stateName + "] " + key + " not exist")
	}

	valueAsSlice, ok := value.([]interface{})
	if !ok {
		var slice []interface{}
		return slice, errors.New("State [" + stateName + "] " + key + " illegal, required []interface{}")
	}
	return valueAsSlice, nil
}

func (b BaseStateParser) GetSliceOrDefault(stateMap map[string]interface{}, key string, defaultValue []interface{}) []interface{} {
	value := stateMap[key]

	if value == nil {
		return defaultValue
	}

	valueAsSlice, ok := value.([]interface{})
	if !ok {
		return defaultValue
	}
	return valueAsSlice
}

func (b BaseStateParser) GetMapOrDefault(stateMap map[string]interface{}, key string, defaultValue map[string]interface{}) map[string]interface{} {
	value := stateMap[key]

	if value == nil {
		return defaultValue
	}

	valueAsMap, ok := value.(map[string]interface{})
	if !ok {
		return defaultValue
	}
	return valueAsMap
}

func (b BaseStateParser) GetBool(stateName string, stateMap map[string]interface{}, key string) (bool, error) {
	value := stateMap[key]

	if value == nil {
		return false, errors.New("State [" + stateName + "] " + key + " not exist")
	}

	valueAsBool, ok := value.(bool)
	if !ok {
		return false, errors.New("State [" + stateName + "] " + key + " illegal, required bool")
	}
	return valueAsBool, nil
}

func (b BaseStateParser) GetBoolOrFalse(stateMap map[string]interface{}, key string) bool {
	value := stateMap[key]

	if value == nil {
		return false
	}

	valueAsBool, ok := value.(bool)
	if !ok {
		return false
	}
	return valueAsBool
}

func (b BaseStateParser) GetIntOrDefault(stateMap map[string]interface{}, key string, defaultValue int) int {
	value := stateMap[key]

	if value == nil {
		return defaultValue
	}

	valueAsInt, ok := value.(int)
	if !ok {
		return defaultValue
	}
	return valueAsInt
}

func (b BaseStateParser) GetFloat64OrDefault(stateMap map[string]interface{}, key string, defaultValue float64) float64 {
	value := stateMap[key]

	if value == nil {
		return defaultValue
	}

	valueAsFloat64, ok := value.(float64)
	if !ok {
		return defaultValue
	}
	return valueAsFloat64
}

type StateParserFactory interface {
	RegistryStateParser(stateType string, stateParser StateParser)

	GetStateParser(stateType string) StateParser
}

type DefaultStateParserFactory struct {
	stateParserMap map[string]StateParser
	mutex          sync.Mutex
}

func NewDefaultStateParserFactory() *DefaultStateParserFactory {
	var stateParserMap map[string]StateParser = make(map[string]StateParser)
	return &DefaultStateParserFactory{
		stateParserMap: stateParserMap,
	}
}

// InitDefaultStateParser init StateParser by default
func (d *DefaultStateParserFactory) InitDefaultStateParser() {
	choiceStateParser := NewChoiceStateParser()
	serviceTaskStateParser := NewServiceTaskStateParser()
	subStateMachineParser := NewSubStateMachineParser()
	succeedEndStateParser := NewSucceedEndStateParser()
	compensationTriggerStateParser := NewCompensationTriggerStateParser()
	failEndStateParser := NewFailEndStateParser()

	d.RegistryStateParser(choiceStateParser.StateType(), choiceStateParser)
	d.RegistryStateParser(serviceTaskStateParser.StateType(), serviceTaskStateParser)
	d.RegistryStateParser(subStateMachineParser.StateType(), subStateMachineParser)
	d.RegistryStateParser(succeedEndStateParser.StateType(), succeedEndStateParser)
	d.RegistryStateParser(compensationTriggerStateParser.StateType(), compensationTriggerStateParser)
	d.RegistryStateParser(compensationTriggerStateParser.StateType(), compensationTriggerStateParser)
	d.RegistryStateParser(failEndStateParser.StateType(), failEndStateParser)
}

func (d *DefaultStateParserFactory) RegistryStateParser(stateType string, stateParser StateParser) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.stateParserMap[stateType] = stateParser
}

func (d *DefaultStateParserFactory) GetStateParser(stateType string) StateParser {
	return d.stateParserMap[stateType]
}
