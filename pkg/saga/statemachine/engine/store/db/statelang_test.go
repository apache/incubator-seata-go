package db

import (
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
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
