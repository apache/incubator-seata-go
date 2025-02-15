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

package repository

import (
	"io"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineRepositoryImpl struct {
}

func (s StateMachineRepositoryImpl) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) RegistryStateMachine(machine statelang.StateMachine) error {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) RegistryStateMachineByReader(reader io.Reader) error {
	//TODO implement me
	panic("implement me")
}
