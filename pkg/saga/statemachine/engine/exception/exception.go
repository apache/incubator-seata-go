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

package exception

import (
	"fmt"
	"github.com/seata/seata-go/pkg/util/errors"
)

type EngineExecutionException struct {
	errors.SeataError
	stateName              string
	stateMachineName       string
	stateMachineInstanceId string
	stateInstanceId        string
	ErrCode                string
}

func (e *EngineExecutionException) Error() string {
	return fmt.Sprintf("EngineExecutionException: %s", e.ErrCode)
}

func NewEngineExecutionException(code errors.TransactionErrorCode, msg string, parent error) *EngineExecutionException {
	seataError := errors.New(code, msg, parent)
	return &EngineExecutionException{
		SeataError: *seataError,
	}
}

func (e *EngineExecutionException) StateName() string {
	return e.stateName
}

func (e *EngineExecutionException) SetStateName(stateName string) {
	e.stateName = stateName
}

func (e *EngineExecutionException) StateMachineName() string {
	return e.stateMachineName
}

func (e *EngineExecutionException) SetStateMachineName(stateMachineName string) {
	e.stateMachineName = stateMachineName
}

func (e *EngineExecutionException) StateMachineInstanceId() string {
	return e.stateMachineInstanceId
}

func (e *EngineExecutionException) SetStateMachineInstanceId(stateMachineInstanceId string) {
	e.stateMachineInstanceId = stateMachineInstanceId
}

func (e *EngineExecutionException) StateInstanceId() string {
	return e.stateInstanceId
}

func (e *EngineExecutionException) SetStateInstanceId(stateInstanceId string) {
	e.stateInstanceId = stateInstanceId
}

type ForwardInvalidException struct {
	EngineExecutionException
}
