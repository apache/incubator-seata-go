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

	state.SetCompensateState(a.GetStringOrDefault(stateMap, "CompensateState", ""))
	state.SetForCompensation(a.GetBoolOrFalse(stateMap, "IsForCompensation"))
	state.SetForUpdate(a.GetBoolOrFalse(stateMap, "IsForUpdate"))
	state.SetPersist(a.GetBoolOrFalse(stateMap, "IsPersist"))
	state.SetRetryPersistModeUpdate(a.GetBoolOrFalse(stateMap, "IsRetryPersistModeUpdate"))
	state.SetCompensatePersistModeUpdate(a.GetBoolOrFalse(stateMap, "IsCompensatePersistModeUpdate"))

	retryInterfaces := a.GetSliceOrDefault(stateMap, "Retry", nil)
	if retryInterfaces != nil {
		retries, err := a.parseRetries(state.Name(), retryInterfaces)
		if err != nil {
			return err
		}
		state.SetRetry(retries)
	}

	catchInterfaces := a.GetSliceOrDefault(stateMap, "Catch", nil)
	if catchInterfaces != nil {
		catches, err := a.parseCatches(state.Name(), catchInterfaces)
		if err != nil {
			return err
		}
		state.SetCatches(catches)
	}

	inputInterfaces := a.GetSliceOrDefault(stateMap, "Input", nil)
	if inputInterfaces != nil {
		state.SetInput(inputInterfaces)
	}

	output := a.GetMapOrDefault(stateMap, "Output", nil)
	if output != nil {
		state.SetOutput(output)
	}

	statusMap, ok := stateMap["Status"].(map[string]string)
	if ok {
		state.SetStatus(statusMap)
	}

	loopMap, ok := stateMap["Loop"].(map[string]interface{})
	if ok {
		loop := a.parseLoop(loopMap)
		state.SetLoop(loop)
	}

	return nil
}

func (a *AbstractTaskStateParser) parseLoop(loopMap map[string]interface{}) state.Loop {
	loopImpl := &state.LoopImpl{}
	loopImpl.SetParallel(a.GetIntOrDefault(loopMap, "Parallel", 1))
	loopImpl.SetCollection(a.GetStringOrDefault(loopMap, "Collection", ""))
	loopImpl.SetElementVariableName(a.GetStringOrDefault(loopMap, "ElementVariableName", "loopElement"))
	loopImpl.SetElementIndexName(a.GetStringOrDefault(loopMap, "ElementIndexName", "loopCounter"))
	loopImpl.SetElementIndexName(a.GetStringOrDefault(loopMap, "CompletionCondition", "[nrOfInstances] == [nrOfCompletedInstances]"))
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
		errorTypes := a.GetSliceOrDefault(retryMap, "Exceptions", nil)
		if errorTypes != nil {
			errorTypeNames := make([]string, 0)
			for _, errorType := range errorTypes {
				errorTypeNames = append(errorTypeNames, errorType.(string))
			}
			retry.SetErrorTypeNames(errorTypeNames)
		}
		retry.SetMaxAttempt(a.GetIntOrDefault(retryMap, "MaxAttempts", 0))
		retry.SetBackoffRate(a.GetFloat64OrDefault(retryMap, "BackoffInterval", 0))
		retry.SetIntervalSecond(a.GetFloat64OrDefault(retryMap, "IntervalSeconds", 0))
		retries = append(retries, retry)
	}
	return retries, nil
}

func (a *AbstractTaskStateParser) parseCatches(stateName string, catchInterfaces []interface{}) ([]state.ErrorMatch, error) {
	errorMatches := make([]state.ErrorMatch, len(catchInterfaces))
	for _, catchInterface := range catchInterfaces {
		catchMap, ok := catchInterface.(map[string]interface{})
		if !ok {
			return nil, errors.New("State [" + stateName + "] " + "Catch illegal, require map[string]interface{}")
		}
		errorMatch := &state.ErrorMatchImpl{}
		errorInterfaces := a.GetSliceOrDefault(catchMap, "Exceptions", nil)
		if errorInterfaces != nil {
			errorNames := make([]string, 0)
			for _, errorType := range errorInterfaces {
				errorNames = append(errorNames, errorType.(string))
			}
			errorMatch.SetErrors(errorNames)
		}
		next := a.GetStringOrDefault(catchMap, "Next", "")
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

	serviceTaskStateImpl.SetServiceType(s.GetStringOrDefault(stateMap, "ServiceType", ""))

	parameterTypeInterfaces := s.GetSliceOrDefault(stateMap, "ParameterTypes", nil)
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

	serviceTaskStateImpl.SetIsAsync(s.GetBoolOrFalse(stateMap, "IsAsync"))

	return serviceTaskStateImpl, nil
}
