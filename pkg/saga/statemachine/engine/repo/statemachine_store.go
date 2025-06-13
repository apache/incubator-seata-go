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

package repo

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"io"
)

type StateLogRepository interface {
	GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateMachineInstance, error)

	GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateInstance, error)

	GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error)

	GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error)
}

type StateMachineRepository interface {
	GetStateMachineById(stateMachineId string) (statelang.StateMachine, error)

	GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	RegistryStateMachine(statelang.StateMachine) error

	RegistryStateMachineByReader(reader io.Reader) error
}
