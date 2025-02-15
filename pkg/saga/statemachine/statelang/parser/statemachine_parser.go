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

package parser

import (
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"strconv"
	"strings"
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

func NewBaseStateParser() *BaseStateParser {
	return &BaseStateParser{}
}

func (b BaseStateParser) ParseBaseAttributes(stateName string, state statelang.State, stateMap map[string]interface{}) error {
	state.SetName(stateName)

	comment, err := b.GetStringOrDefault(stateName, stateMap, "Comment", "")
	if err != nil {
		return err
	}
	state.SetComment(comment)

	next, err := b.GetStringOrDefault(stateName, stateMap, "Next", "")
	if err != nil {
		return err
	}
	state.SetNext(next)
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

func (b BaseStateParser) GetStringOrDefault(stateName string, stateMap map[string]interface{}, key string, defaultValue string) (string, error) {
	value := stateMap[key]
	if value == nil {
		return defaultValue, nil
	}

	valueAsString, ok := value.(string)
	if !ok {
		return defaultValue, errors.New("State [" + stateName + "] " + key + " illegal, required string")
	}
	return valueAsString, nil
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

func (b BaseStateParser) GetSliceOrDefault(stateName string, stateMap map[string]interface{}, key string, defaultValue []interface{}) ([]interface{}, error) {
	value := stateMap[key]

	if value == nil {
		return defaultValue, nil
	}

	valueAsSlice, ok := value.([]interface{})
	if !ok {
		return defaultValue, errors.New("State [" + stateName + "] " + key + " illegal, required []interface{}")
	}
	return valueAsSlice, nil
}

func (b BaseStateParser) GetMapOrDefault(stateMap map[string]interface{}, key string, defaultValue map[string]interface{}) (map[string]interface{}, error) {
	value := stateMap[key]

	if value == nil {
		return defaultValue, nil
	}

	valueAsMap, ok := value.(map[string]interface{})
	if !ok {
		return defaultValue, nil
	}
	return valueAsMap, nil
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

func (b BaseStateParser) GetBoolOrDefault(stateName string, stateMap map[string]interface{}, key string, defaultValue bool) (bool, error) {
	value := stateMap[key]

	if value == nil {
		return defaultValue, nil
	}

	valueAsBool, ok := value.(bool)
	if !ok {
		return defaultValue, errors.New("State [" + stateName + "] " + key + " illegal, required bool")
	}
	return valueAsBool, nil
}

func (b BaseStateParser) GetIntOrDefault(stateName string, stateMap map[string]interface{}, key string, defaultValue int) (int, error) {
	value := stateMap[key]

	if value == nil {
		return defaultValue, nil
	}

	// use float64 conversion when the configuration file is json, and use int conversion when the configuration file is yaml
	valueAsFloat64, okToFloat64 := value.(float64)
	valueAsInt, okToInt := value.(int)
	if !okToFloat64 && !okToInt {
		return defaultValue, errors.New("State [" + stateName + "] " + key + " illegal, required int")
	}

	if okToFloat64 {
		floatStr := strconv.FormatFloat(valueAsFloat64, 'f', -1, 64)
		if strings.Contains(floatStr, ".") {
			return defaultValue, errors.New("State [" + stateName + "] " + key + " illegal, required int")
		}

		return int(valueAsFloat64), nil
	}

	return valueAsInt, nil
}

func (b BaseStateParser) GetFloat64OrDefault(stateName string, stateMap map[string]interface{}, key string, defaultValue float64) (float64, error) {
	value := stateMap[key]

	if value == nil {
		return defaultValue, nil
	}

	// use float64 conversion when the configuration file is json, and use int conversion when the configuration file is yaml
	valueAsFloat64, okToFloat64 := value.(float64)
	valueAsInt, okToInt := value.(int)
	if !okToFloat64 && !okToInt {
		return defaultValue, errors.New("State [" + stateName + "] " + key + " illegal, required float64")
	}

	if okToFloat64 {
		return valueAsFloat64, nil
	}
	return float64(valueAsInt), nil
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
	scriptTaskStateParser := NewScriptTaskStateParser()

	d.RegistryStateParser(choiceStateParser.StateType(), choiceStateParser)
	d.RegistryStateParser(serviceTaskStateParser.StateType(), serviceTaskStateParser)
	d.RegistryStateParser(subStateMachineParser.StateType(), subStateMachineParser)
	d.RegistryStateParser(succeedEndStateParser.StateType(), succeedEndStateParser)
	d.RegistryStateParser(compensationTriggerStateParser.StateType(), compensationTriggerStateParser)
	d.RegistryStateParser(compensationTriggerStateParser.StateType(), compensationTriggerStateParser)
	d.RegistryStateParser(failEndStateParser.StateType(), failEndStateParser)
	d.RegistryStateParser(scriptTaskStateParser.StateType(), scriptTaskStateParser)
}

func (d *DefaultStateParserFactory) RegistryStateParser(stateType string, stateParser StateParser) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.stateParserMap[stateType] = stateParser
}

func (d *DefaultStateParserFactory) GetStateParser(stateType string) StateParser {
	return d.stateParserMap[stateType]
}
