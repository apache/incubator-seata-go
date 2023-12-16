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

	comment, err := GetString(stateName, stateMap, "Comment")
	if err != nil {
		return err
	}
	state.SetComment(comment)

	next, err := GetString(stateName, stateMap, "Next")
	if err != nil {
		return err
	}

	state.SetNext(next)
	return nil
}

func GetString(stateName string, stateMap map[string]interface{}, key string) (string, error) {
	value, ok := stateMap[key].(string)
	if !ok {
		var s string
		return s, errors.New("State [" + stateName + "] " + key + " illegal, required string")
	}
	return value, nil
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

	d.RegistryStateParser(choiceStateParser.StateType(), choiceStateParser)
}

func (d *DefaultStateParserFactory) RegistryStateParser(stateType string, stateParser StateParser) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.stateParserMap[stateType] = stateParser
}

func (d *DefaultStateParserFactory) GetStateParser(stateType string) StateParser {
	return d.stateParserMap[stateType]
}
