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

package db

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestStoreAndGetStateMachine(t *testing.T) {
	prepareDB()

	stateLangStore := NewStateLangStore(db, "seata_")
	const stateMachineId = "simpleStateMachine"
	expected := statelang.NewStateMachineImpl()
	expected.SetID(stateMachineId)
	expected.SetName("simpleStateMachine")
	expected.SetComment("This is a test state machine")
	expected.SetCreateTime(time.Now())
	err := stateLangStore.StoreStateMachine(expected)
	if err != nil {
		t.Error(err)
		return
	}

	actual, err := stateLangStore.GetStateMachineById(stateMachineId)
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, expected.ID(), actual.ID())
	assert.Equal(t, expected.Name(), actual.Name())
	assert.Equal(t, expected.Comment(), actual.Comment())
	assert.Equal(t, expected.CreateTime().UnixNano(), actual.CreateTime().UnixNano())
}

func TestStoreAndGetLastVersionStateMachine(t *testing.T) {
	prepareDB()

	stateLangStore := NewStateLangStore(db, "seata_")
	const stateMachineName, tenantId = "simpleStateMachine", "test"
	stateMachineV1 := statelang.NewStateMachineImpl()
	stateMachineV1.SetID("simpleStateMachineV1")
	stateMachineV1.SetName(stateMachineName)
	stateMachineV1.SetTenantId(tenantId)
	stateMachineV1.SetCreateTime(time.Now().Add(time.Duration(-1) * time.Millisecond))

	stateMachineV2 := statelang.NewStateMachineImpl()
	stateMachineV2.SetID("simpleStateMachineV2")
	stateMachineV2.SetName(stateMachineName)
	stateMachineV2.SetTenantId(tenantId)
	stateMachineV2.SetCreateTime(time.Now())

	err := stateLangStore.StoreStateMachine(stateMachineV1)
	if err != nil {
		t.Error(err)
		return
	}
	err = stateLangStore.StoreStateMachine(stateMachineV2)
	if err != nil {
		t.Error(err)
		return
	}

	actual, err := stateLangStore.GetLastVersionStateMachine(stateMachineName, tenantId)
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, stateMachineV2.ID(), actual.ID())
}
