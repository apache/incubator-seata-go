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
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/parser"
	"github.com/seata/seata-go/pkg/saga/statemachine/store/db"
)

var (
	oncePrepareDB sync.Once
	testdb        *sql.DB
)

func prepareDB() {
	oncePrepareDB.Do(func() {
		var err error
		testdb, err = sql.Open("sqlite3", ":memory:")
		if err != nil {
			panic(err)
		}
		initScript, err := readInitSQL()
		if err != nil {
			panic(err)
		}
		if _, err := testdb.Exec(initScript); err != nil {
			panic(err)
		}
	})
}

func readInitSQL() (string, error) {
	return readTestFile("testdata/sql/saga/sqlite_init.sql")
}

func loadStateMachineByYaml() string {
	path, err := locateTestFile("testdata/saga/statelang/simple_statemachine.json")
	if err != nil {
		panic(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func readTestFile(rel string) (string, error) {
	path, err := locateTestFile(rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func locateTestFile(rel string) (string, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	base := filepath.Dir(thisFile)
	candidates := []string{
		filepath.Join(base, rel),
		filepath.Join(base, "../../../../../", rel),
		filepath.Join(base, "../../../../../../", rel),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("test file not found: %s (checked %v)", rel, candidates)
}

func TestStateMachineInMemory(t *testing.T) {
	const stateMachineId, stateMachineName, tenantId = "simpleStateMachine", "simpleStateMachine", "test"
	stateMachine := statelang.NewStateMachineImpl()
	stateMachine.SetID(stateMachineId)
	stateMachine.SetName(stateMachineName)
	stateMachine.SetTenantId(tenantId)
	stateMachine.SetComment("This is a test state machine")
	stateMachine.SetCreateTime(time.Now())

	repository := GetStateMachineRepositoryImpl()

	err := repository.RegistryStateMachine(stateMachine)
	assert.Nil(t, err)

	machineById, err := repository.GetStateMachineById(stateMachine.ID())
	assert.Nil(t, err)
	assert.Equal(t, stateMachine.Name(), machineById.Name())
	assert.Equal(t, stateMachine.TenantId(), machineById.TenantId())
	assert.Equal(t, stateMachine.Comment(), machineById.Comment())
	assert.Equal(t, stateMachine.CreateTime().UnixNano(), machineById.CreateTime().UnixNano())

	machineByNameAndTenantId, err := repository.GetLastVersionStateMachine(stateMachine.Name(), stateMachine.TenantId())
	assert.Nil(t, err)
	assert.Equal(t, stateMachine.ID(), machineByNameAndTenantId.ID())
	assert.Equal(t, stateMachine.Comment(), machineById.Comment())
	assert.Equal(t, stateMachine.CreateTime().UnixNano(), machineById.CreateTime().UnixNano())
}

func TestStateMachineInDb(t *testing.T) {
	prepareDB()

	const tenantId = "test"
	yaml := loadStateMachineByYaml()
	stateMachine, err := parser.NewJSONStateMachineParser().Parse(yaml)
	assert.Nil(t, err)
	stateMachine.SetTenantId(tenantId)
	stateMachine.SetContent(yaml)

	repository := GetStateMachineRepositoryImpl()
	repository.SetStateLangStore(db.NewStateLangStore(testdb, "seata_"))

	err = repository.RegistryStateMachine(stateMachine)
	assert.Nil(t, err)

	repository.stateMachineMapById[stateMachine.ID()] = nil
	machineById, err := repository.GetStateMachineById(stateMachine.ID())
	assert.Nil(t, err)
	assert.Equal(t, stateMachine.Name(), machineById.Name())
	assert.Equal(t, stateMachine.TenantId(), machineById.TenantId())
	assert.Equal(t, stateMachine.Comment(), machineById.Comment())
	assert.Equal(t, stateMachine.CreateTime().UnixNano(), machineById.CreateTime().UnixNano())

	repository.stateMachineMapByNameAndTenant[stateMachine.Name()+"_"+stateMachine.TenantId()] = nil
	machineByNameAndTenantId, err := repository.GetLastVersionStateMachine(stateMachine.Name(), stateMachine.TenantId())
	assert.Nil(t, err)
	assert.Equal(t, stateMachine.ID(), machineByNameAndTenantId.ID())
	assert.Equal(t, stateMachine.Comment(), machineById.Comment())
	assert.Equal(t, stateMachine.CreateTime().UnixNano(), machineById.CreateTime().UnixNano())
}
