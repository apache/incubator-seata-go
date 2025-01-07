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
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type AbstractTaskStateParser struct {
	*BaseStateParser
}

func NewAbstractTaskStateParser() *AbstractTaskStateParser {
	return &AbstractTaskStateParser{
		&BaseStateParser{},
	}
}

func (a *AbstractTaskStateParser) ParseTaskAttributes(stateName string, state *state.AbstractTaskState, stateMap map[string]interface{}) error {
	err := a.ParseBaseAttributes(state.Name(), state.BaseState, stateMap)
	if err != nil {
		return err
	}

	compensateState, err := a.GetStringOrDefault(stateName, stateMap, "CompensateState", "")
	if err != nil {
		return err
	}
	state.SetCompensateState(compensateState)

	isForCompensation, err := a.GetBoolOrDefault(stateName, stateMap, "IsForCompensation", false)
	if err != nil {
		return err
	}
	state.SetForCompensation(isForCompensation)

	isForUpdate, err := a.GetBoolOrDefault(stateName, stateMap, "IsForUpdate", false)
	if err != nil {
		return err
	}
	state.SetForUpdate(isForUpdate)

	isPersist, err := a.GetBoolOrDefault(stateName, stateMap, "IsPersist", false)
	if err != nil {
		return err
	}
	state.SetPersist(isPersist)

	isRetryPersistModeUpdate, err := a.GetBoolOrDefault(stateName, stateMap, "IsRetryPersistModeUpdate", false)
	if err != nil {
		return err
	}
	state.SetRetryPersistModeUpdate(isRetryPersistModeUpdate)

	isCompensatePersistModeUpdate, err := a.GetBoolOrDefault(stateName, stateMap, "IsCompensatePersistModeUpdate", false)
	if err != nil {
		return err
	}
	state.SetCompensatePersistModeUpdate(isCompensatePersistModeUpdate)

	retryInterfaces, err := a.GetSliceOrDefault(stateName, stateMap, "Retry", nil)
	if err != nil {
		return err
	}
	if retryInterfaces != nil {
		retries, err := a.parseRetries(state.Name(), retryInterfaces)
		if err != nil {
			return err
		}
		state.SetRetry(retries)
	}

	catchInterfaces, err := a.GetSliceOrDefault(stateName, stateMap, "Catch", nil)
	if err != nil {
		return err
	}
	if catchInterfaces != nil {
		catches, err := a.parseCatches(state.Name(), catchInterfaces)
		if err != nil {
			return err
		}
		state.SetCatches(catches)
	}

	inputInterfaces, err := a.GetSliceOrDefault(stateName, stateMap, "Input", nil)
	if err != nil {
		return err
	}
	if inputInterfaces != nil {
		state.SetInput(inputInterfaces)
	}

	output, err := a.GetMapOrDefault(stateMap, "Output", nil)
	if err != nil {
		return err
	}
	if output != nil {
		state.SetOutput(output)
	}

	statusMap, ok := stateMap["Status"].(map[string]string)
	if ok {
		state.SetStatus(statusMap)
	}

	loopMap, ok := stateMap["Loop"].(map[string]interface{})
	if ok {
		loop := a.parseLoop(stateName, loopMap)
		state.SetLoop(loop)
	}

	return nil
}

func (a *AbstractTaskStateParser) parseLoop(stateName string, loopMap map[string]interface{}) state.Loop {
	loopImpl := &state.LoopImpl{}
	parallel, err := a.GetIntOrDefault(stateName, loopMap, "Parallel", 1)
	if err != nil {
		return nil
	}
	loopImpl.SetParallel(parallel)

	collection, err := a.GetStringOrDefault(stateName, loopMap, "Collection", "")
	if err != nil {
		return nil
	}
	loopImpl.SetCollection(collection)

	elementVariableName, err := a.GetStringOrDefault(stateName, loopMap, "ElementVariableName", "loopElement")
	if err != nil {
		return nil
	}
	loopImpl.SetElementVariableName(elementVariableName)

	elementIndexName, err := a.GetStringOrDefault(stateName, loopMap, "ElementIndexName", "loopCounter")
	if err != nil {
		return nil
	}
	loopImpl.SetElementIndexName(elementIndexName)

	completionCondition, err := a.GetStringOrDefault(stateName, loopMap, "CompletionCondition", "[nrOfInstances] == [nrOfCompletedInstances]")
	if err != nil {
		return nil
	}
	loopImpl.SetElementIndexName(completionCondition)
	return loopImpl
}

func (a *AbstractTaskStateParser) parseRetries(stateName string, retryInterfaces []interface{}) ([]state.Retry, error) {
	retries := make([]state.Retry, 0)
	for _, retryInterface := range retryInterfaces {
		retryMap, ok := retryInterface.(map[string]interface{})
		if !ok {

			return nil, errors.New("State [" + stateName + "] " + "Retry illegal， require map[string]interface{}")
		}
		retry := &state.RetryImpl{}
		exceptions, err := a.GetSliceOrDefault(stateName, retryMap, "Exceptions", nil)
		if err != nil {
			return nil, err
		}
		if exceptions != nil {
			errors := make([]string, 0)
			for _, errorType := range exceptions {
				errors = append(errors, errorType.(string))
			}
			retry.SetExceptions(errors)
		}

		maxAttempts, err := a.GetIntOrDefault(stateName, retryMap, "MaxAttempts", 0)
		if err != nil {
			return nil, err
		}
		retry.SetMaxAttempt(maxAttempts)

		backoffInterval, err := a.GetFloat64OrDefault(stateName, retryMap, "BackoffInterval", 0)
		if err != nil {
			return nil, err
		}
		retry.SetBackoffRate(backoffInterval)

		intervalSeconds, err := a.GetFloat64OrDefault(stateName, retryMap, "IntervalSeconds", 0)
		if err != nil {
			return nil, err
		}
		retry.SetIntervalSecond(intervalSeconds)
		retries = append(retries, retry)
	}
	return retries, nil
}

func (a *AbstractTaskStateParser) parseCatches(stateName string, catchInterfaces []interface{}) ([]state.ExceptionMatch, error) {
	errorMatches := make([]state.ExceptionMatch, 0, len(catchInterfaces))
	for _, catchInterface := range catchInterfaces {
		catchMap, ok := catchInterface.(map[string]interface{})
		if !ok {
			return nil, errors.New("State [" + stateName + "] " + "Catch illegal, require map[string]interface{}")
		}
		errorMatch := &state.ExceptionMatchImpl{}
		errorInterfaces, err := a.GetSliceOrDefault(stateName, catchMap, "Exceptions", nil)
		if err != nil {
			return nil, err
		}
		if errorInterfaces != nil {
			errorNames := make([]string, 0)
			for _, errorType := range errorInterfaces {
				errorNames = append(errorNames, errorType.(string))
			}
			errorMatch.SetExceptions(errorNames)
		}
		next, err := a.GetStringOrDefault(stateName, catchMap, "Next", "")
		if err != nil {
			return nil, err
		}
		errorMatch.SetNext(next)
		errorMatches = append(errorMatches, errorMatch)
	}
	return errorMatches, nil
}

type ServiceTaskStateParser struct {
	*AbstractTaskStateParser
}

func NewServiceTaskStateParser() *ServiceTaskStateParser {
	return &ServiceTaskStateParser{
		NewAbstractTaskStateParser(),
	}
}

func (s ServiceTaskStateParser) StateType() string {
	return constant.StateTypeServiceTask
}

func (s ServiceTaskStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()

	err := s.ParseTaskAttributes(stateName, serviceTaskStateImpl.AbstractTaskState, stateMap)
	if err != nil {
		return nil, err
	}

	serviceName, err := s.GetString(stateName, stateMap, "ServiceName")
	if err != nil {
		return nil, err
	}
	serviceTaskStateImpl.SetServiceName(serviceName)

	serviceMethod, err := s.GetString(stateName, stateMap, "ServiceMethod")
	if err != nil {
		return nil, err
	}
	serviceTaskStateImpl.SetServiceMethod(serviceMethod)

	serviceType, err := s.GetStringOrDefault(stateName, stateMap, "ServiceType", "")
	if err != nil {
		return nil, err
	}
	serviceTaskStateImpl.SetServiceType(serviceType)

	parameterTypeInterfaces, err := s.GetSliceOrDefault(stateName, stateMap, "ParameterTypes", nil)
	if err != nil {
		return nil, err
	}
	if parameterTypeInterfaces != nil {
		var parameterTypes []string
		for i := range parameterTypeInterfaces {
			parameterType, ok := parameterTypeInterfaces[i].(string)
			if !ok {
				return nil, errors.New(fmt.Sprintf("State [%s] parameterType required string", stateName))
			}

			parameterTypes = append(parameterTypes, parameterType)
		}
		serviceTaskStateImpl.SetParameterTypes(parameterTypes)
	}

	isAsync, err := s.GetBoolOrDefault(stateName, stateMap, "IsAsync", false)
	if err != nil {
		return nil, err
	}
	serviceTaskStateImpl.SetIsAsync(isAsync)

	return serviceTaskStateImpl, nil
}

type ScriptTaskStateParser struct {
	*AbstractTaskStateParser
}

func NewScriptTaskStateParser() *ScriptTaskStateParser {
	return &ScriptTaskStateParser{
		NewAbstractTaskStateParser(),
	}
}

func (s ScriptTaskStateParser) StateType() string {
	return constant.StateTypeScriptTask
}

func (s ScriptTaskStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	scriptTaskStateImpl := state.NewScriptTaskStateImpl()

	err := s.ParseTaskAttributes(stateName, scriptTaskStateImpl.AbstractTaskState, stateMap)
	if err != nil {
		return nil, err
	}

	scriptType, err := s.GetStringOrDefault(stateName, stateMap, "ScriptType", "")
	if err != nil {
		return nil, err
	}
	if scriptType != "" {
		scriptTaskStateImpl.SetScriptType(scriptType)
	}

	scriptContent, err := s.GetStringOrDefault(stateName, stateMap, "ScriptContent", "")
	if err != nil {
		return nil, err
	}
	scriptTaskStateImpl.SetScriptContent(scriptContent)

	scriptTaskStateImpl.SetForCompensation(false)
	scriptTaskStateImpl.SetForUpdate(false)
	scriptTaskStateImpl.SetPersist(false)

	return scriptTaskStateImpl, nil
}
