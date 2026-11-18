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
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang/parser"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/store/db"
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
	// go-sqlite3 needs cgo: in CGO_ENABLED=0 builds the stub driver panics and
	// aborts the whole test binary, which would also prevent every later test
	// in this package from running. Skip cleanly instead.
	if !sqliteAvailable() {
		t.Skip("sqlite3 driver requires cgo; skipping in CGO_ENABLED=0 builds")
	}
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

// TestStateMachineRepository_ConcurrentAccess exercises concurrent registry
// writes (map writes) mixed with lookups (map reads); before the RWMutex fix
// the concurrent writes fatal'd with "concurrent map writes" (this test
// fails on the pre-fix code).
func TestStateMachineRepository_ConcurrentAccess(t *testing.T) {
	json := loadStateMachineByYaml()
	assert.NotEmpty(t, json)

	repo := GetStateMachineRepositoryImpl()
	// Parse machines up front: the JSON parser is not what is under test here.
	machines := make([]statelang.StateMachine, 32)
	for i := 0; i < 32; i++ {
		sm, err := parser.NewJSONStateMachineParser().Parse(json)
		assert.NoError(t, err)
		sm.SetID(fmt.Sprintf("concurrent-sm-%d", i))
		sm.SetName("concurrent_state_machine")
		sm.SetTenantId(fmt.Sprintf("tenant-%d", i%4))
		sm.SetContent(json)
		machines[i] = sm
	}

	var wg sync.WaitGroup
	errs := make(chan error, 128)
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				slot := (worker + i) % 32
				// Concurrent registry writes: without the RWMutex these are
				// unsynchronized map writes.
				if err := repo.RegistryStateMachine(machines[slot]); err != nil {
					errs <- err
					return
				}
				id := fmt.Sprintf("concurrent-sm-%d", slot)
				if _, err := repo.GetStateMachineById(id); err != nil {
					errs <- err
					return
				}
				if _, err := repo.GetLastVersionStateMachine("concurrent_state_machine", fmt.Sprintf("tenant-%d", slot%4)); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err)
	}
}

// stubStateLangStore is a minimal StateLangStore used to exercise the
// cold-fill path without a database (the sqlite-backed tests require cgo).
type stubStateLangStore struct {
	// lastVersionCalls counts GetLastVersionStateMachine invocations.
	lastVersionCalls atomic.Int64
	// firstCallDelay, when positive, is slept on the first store call so that
	// concurrent lookup goroutines all observe the cache miss before the
	// first fill completes (mimics slow DB I/O under the fill lock).
	firstCallDelay time.Duration
	machine        statelang.StateMachine
}

func (s *stubStateLangStore) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	return nil, nil
}

func (s *stubStateLangStore) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	n := s.lastVersionCalls.Add(1)
	if n == 1 && s.firstCallDelay > 0 {
		time.Sleep(s.firstCallDelay)
	}
	return s.machine, nil
}

func (s *stubStateLangStore) StoreStateMachine(stateMachine statelang.StateMachine) error {
	return nil
}

// TestGetLastVersionStateMachine_CachesUnderStoreLock verifies the
// double-check consults the name_tenant keyed map. The first section proves
// the name_tenant cache is served without a second store trip even when the
// ById map is empty; the concurrent section proves that concurrent cold
// lookups of the same key hit the store exactly once, because the in-lock
// double check now consults the name_tenant map the first caller filled
// (previously it checked stateMachineMapById with a name_tenant key and
// always missed, so every waiter went back to the store).
func TestGetLastVersionStateMachine_CachesUnderStoreLock(t *testing.T) {
	json := loadStateMachineByYaml()
	assert.NotEmpty(t, json)

	sm, err := parser.NewJSONStateMachineParser().Parse(json)
	assert.NoError(t, err)
	sm.SetName("cache_probe_state_machine")
	sm.SetTenantId("default")
	sm.SetContent(json)

	repo := GetStateMachineRepositoryImpl()
	// The repository is a process-wide singleton shared by other tests (and
	// by repeated -count runs): start from empty caches.
	repo.stateMachineMapById = map[string]statelang.StateMachine{}
	repo.stateMachineMapByNameAndTenant = map[string]statelang.StateMachine{}
	stub := &stubStateLangStore{machine: sm}
	repo.SetStateLangStore(stub)
	defer repo.SetStateLangStore(nil)

	// First lookup cold-fills both maps from the store.
	_, err = repo.GetLastVersionStateMachine("cache_probe_state_machine", "default")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), stub.lastVersionCalls.Load())

	// Second lookup must be served from the name_tenant cache even though the
	// ById map was wiped: the store is still consulted exactly once.
	repo.stateMachineMapById = map[string]statelang.StateMachine{}
	_, err = repo.GetLastVersionStateMachine("cache_probe_state_machine", "default")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), stub.lastVersionCalls.Load())

	// Concurrent cold lookups of the same key: the first caller fills under
	// the write lock, the waiters must be served by the in-lock double check
	// against the name_tenant map, so the store is consulted exactly once.
	// (The first store call is delayed to make sure every goroutine observes
	// the miss and queues on the fill lock before the fill completes.)
	stub.lastVersionCalls.Store(0)
	stub.firstCallDelay = 100 * time.Millisecond
	repo.stateMachineMapById = map[string]statelang.StateMachine{}
	repo.stateMachineMapByNameAndTenant = map[string]statelang.StateMachine{}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := repo.GetLastVersionStateMachine("cache_probe_state_machine", "default")
			assert.NoError(t, err)
			assert.NotNil(t, got)
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, int64(1), stub.lastVersionCalls.Load())
}
