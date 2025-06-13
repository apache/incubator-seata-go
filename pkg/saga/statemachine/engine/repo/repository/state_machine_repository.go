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
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/parser"
	"github.com/seata/seata-go/pkg/saga/statemachine/store"
	"github.com/seata/seata-go/pkg/util/log"
)

const (
	DefaultJsonParser = "fastjson"
)

var (
	stateMachineRepositoryImpl     *StateMachineRepositoryImpl
	onceStateMachineRepositoryImpl sync.Once
)

type StateMachineRepositoryImpl struct {
	stateMachineMapById            map[string]statelang.StateMachine
	stateMachineMapByNameAndTenant map[string]statelang.StateMachine

	stateLangStore  store.StateLangStore
	seqGenerator    sequence.SeqGenerator
	defaultTenantId string
	jsonParserName  string
	charset         string
	mutex           *sync.Mutex
}

func GetStateMachineRepositoryImpl() *StateMachineRepositoryImpl {
	if stateMachineRepositoryImpl == nil {
		onceStateMachineRepositoryImpl.Do(func() {
			//TODO charset is not use
			//TODO using json parser
			stateMachineRepositoryImpl = &StateMachineRepositoryImpl{
				stateMachineMapById:            make(map[string]statelang.StateMachine),
				stateMachineMapByNameAndTenant: make(map[string]statelang.StateMachine),
				seqGenerator:                   sequence.NewUUIDSeqGenerator(),
				jsonParserName:                 DefaultJsonParser,
				charset:                        "UTF-8",
				mutex:                          &sync.Mutex{},
			}
		})
	}

	return stateMachineRepositoryImpl
}

func (s *StateMachineRepositoryImpl) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	stateMachine := s.stateMachineMapById[stateMachineId]
	if stateMachine == nil && s.stateLangStore != nil {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		stateMachine = s.stateMachineMapById[stateMachineId]
		if stateMachine == nil {
			oldStateMachine, err := s.stateLangStore.GetStateMachineById(stateMachineId)
			if err != nil {
				return oldStateMachine, err
			}

			parseStatMachine, err := parser.NewJSONStateMachineParser().Parse(oldStateMachine.Content())
			if err != nil {
				return oldStateMachine, err
			}

			oldStateMachine.SetStartState(parseStatMachine.StartState())
			for key, val := range parseStatMachine.States() {
				oldStateMachine.States()[key] = val
			}

			s.stateMachineMapById[stateMachineId] = oldStateMachine
			s.stateMachineMapByNameAndTenant[oldStateMachine.Name()+"_"+oldStateMachine.TenantId()] = oldStateMachine
			return oldStateMachine, nil
		}
	}
	return stateMachine, nil
}

func (s *StateMachineRepositoryImpl) GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return s.GetLastVersionStateMachine(stateMachineName, tenantId)
}

func (s *StateMachineRepositoryImpl) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	key := stateMachineName + "_" + tenantId
	stateMachine := s.stateMachineMapByNameAndTenant[key]
	if stateMachine == nil && s.stateLangStore != nil {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		stateMachine = s.stateMachineMapById[key]
		if stateMachine == nil {
			oldStateMachine, err := s.stateLangStore.GetLastVersionStateMachine(stateMachineName, tenantId)
			if err != nil {
				return oldStateMachine, err
			}

			parseStatMachine, err := parser.NewJSONStateMachineParser().Parse(oldStateMachine.Content())
			if err != nil {
				return oldStateMachine, err
			}

			oldStateMachine.SetStartState(parseStatMachine.StartState())
			for key, val := range parseStatMachine.States() {
				oldStateMachine.States()[key] = val
			}

			s.stateMachineMapById[oldStateMachine.ID()] = oldStateMachine
			s.stateMachineMapByNameAndTenant[key] = oldStateMachine
			return oldStateMachine, nil
		}
	}
	return stateMachine, nil
}

func (s *StateMachineRepositoryImpl) RegistryStateMachine(machine statelang.StateMachine) error {
	stateMachineName := machine.Name()
	tenantId := machine.TenantId()

	if s.stateLangStore != nil {
		oldStateMachine, err := s.stateLangStore.GetLastVersionStateMachine(stateMachineName, tenantId)
		if err != nil {
			return err
		}

		if oldStateMachine != nil {
			if oldStateMachine.Content() == machine.Content() && machine.Version() != "" && machine.Version() == oldStateMachine.Version() {
				log.Debugf("StateMachine[%s] is already exist a same version", stateMachineName)
				machine.SetID(oldStateMachine.ID())
				machine.SetCreateTime(oldStateMachine.CreateTime())

				s.stateMachineMapById[machine.ID()] = machine
				s.stateMachineMapByNameAndTenant[machine.Name()+"_"+machine.TenantId()] = machine
				return nil
			}
		}

		if machine.ID() == "" {
			machine.SetID(s.seqGenerator.GenerateId(constant.SeqEntityStateMachine, ""))
		}

		machine.SetCreateTime(time.Now())

		err = s.stateLangStore.StoreStateMachine(machine)
		if err != nil {
			return err
		}
	}

	if machine.ID() == "" {
		machine.SetID(s.seqGenerator.GenerateId(constant.SeqEntityStateMachine, ""))
	}

	s.stateMachineMapById[machine.ID()] = machine
	s.stateMachineMapByNameAndTenant[machine.Name()+"_"+machine.TenantId()] = machine
	return nil
}

func (s *StateMachineRepositoryImpl) RegistryStateMachineByReader(reader io.Reader) error {
	jsonByte, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	json := string(jsonByte)
	parseStatMachine, err := parser.NewJSONStateMachineParser().Parse(json)
	if err != nil {
		return err
	}

	if parseStatMachine == nil {
		return nil
	}

	parseStatMachine.SetContent(json)
	s.RegistryStateMachine(parseStatMachine)

	log.Debugf("===== StateMachine Loaded: %s", json)

	return nil
}

func (s *StateMachineRepositoryImpl) SetStateLangStore(stateLangStore store.StateLangStore) {
	s.stateLangStore = stateLangStore
}

func (s *StateMachineRepositoryImpl) SetSeqGenerator(seqGenerator sequence.SeqGenerator) {
	s.seqGenerator = seqGenerator
}

func (s *StateMachineRepositoryImpl) SetCharset(charset string) {
	s.charset = charset
}

func (s *StateMachineRepositoryImpl) GetCharset() string {
	return s.charset
}

func (s *StateMachineRepositoryImpl) SetDefaultTenantId(defaultTenantId string) {
	s.defaultTenantId = defaultTenantId
}

func (s *StateMachineRepositoryImpl) GetDefaultTenantId() string {
	return s.defaultTenantId
}

func (s *StateMachineRepositoryImpl) SetJsonParserName(jsonParserName string) {
	s.jsonParserName = jsonParserName
}

func (s *StateMachineRepositoryImpl) GetJsonParserName() string {
	return s.jsonParserName
}
