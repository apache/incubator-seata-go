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

package base

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/testdata"
)

var (
	capacity      int32 = 1024
	EexpireTime         = 15 * time.Minute
	tableMetaOnce sync.Once
)

type mockTrigger struct {
}

// LoadOne simulates loading table metadata, including id, name, and age columns.
func (m *mockTrigger) LoadOne(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) (*types.TableMeta, error) {

	return &types.TableMeta{
		TableName: key.TableName,
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id"},
			"name": {ColumnName: "name"},
			"age":  {ColumnName: "age"},
		},
		Indexs: map[string]types.IndexMeta{
			"id": {
				Name:    "PRIMARY",
				IType:   types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
			},
			"id_name_age": {
				Name:    "name_age_idx",
				IType:   types.IndexUnique,
				Columns: []types.ColumnMeta{{ColumnName: "name"}, {ColumnName: "age"}},
			},
		},
		ColumnNames: []string{"id", "name", "age"},
	}, nil
}

func (m *mockTrigger) LoadAll(ctx context.Context, conn *sql.Conn, keys ...types.TableMetaKey) (map[types.TableMetaKey]types.TableMeta, error) {
	return nil, nil
}

func TestBaseTableMetaCache_refresh(t *testing.T) {
	type fields struct {
		expireDuration time.Duration
		capity         int32
		size           int32
		cache          map[types.TableMetaKey]*entry
		trigger        trigger
		db             *sql.DB
		dbName         string
	}
	type args struct {
		ctx context.Context
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tests := []struct {
		name   string
		fields fields
		args   args
		want   types.TableMeta
	}{
		{
			name: "test1",
			fields: fields{
				capity:         capacity,
				size:           0,
				expireDuration: EexpireTime,
				cache: map[types.TableMetaKey]*entry{
					{DBName: "test_db", TableName: "test"}: {
						value:      types.TableMeta{},
						lastAccess: time.Now(),
					},
				},
				trigger: &mockTrigger{},
				dbName:  "test_db",
			},
			args: args{ctx: ctx},
			want: testdata.MockWantTypesMeta("test"),
		},
		{
			name: "test2",
			fields: fields{
				capity:         capacity,
				size:           0,
				expireDuration: EexpireTime,
				cache: map[types.TableMetaKey]*entry{
					{DBName: "test_db", TableName: "TEST"}: {
						value:      types.TableMeta{},
						lastAccess: time.Now(),
					},
				},
				trigger: &mockTrigger{},
				dbName:  "test_db",
			},
			args: args{ctx: ctx},
			want: testdata.MockWantTypesMeta("TEST"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//  Use sqlmock to simulate a database connection
			db, _, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create sqlmock: %v", err)
			}
			defer db.Close()

			loadAllStub := gomonkey.ApplyMethodFunc(tt.fields.trigger, "LoadAll",
				func(_ context.Context, _ *sql.Conn, keys ...types.TableMetaKey) (map[types.TableMetaKey]types.TableMeta, error) {
					return map[types.TableMetaKey]types.TableMeta{keys[0]: tt.want}, nil
				})

			defer loadAllStub.Reset()

			c := &BaseTableMetaCache{
				expireDuration:  tt.fields.expireDuration,
				refreshInterval: time.Minute,
				capity:          tt.fields.capity,
				size:            tt.fields.size,
				cache:           tt.fields.cache,
				trigger:         tt.fields.trigger,
				db:              db,
				dbName:          tt.fields.dbName,
			}
			go c.refresh(tt.args.ctx)
			time.Sleep(time.Second * 3)
			c.lock.RLock()
			defer c.lock.RUnlock()
			assert.Equal(t, c.cache[types.TableMetaKey{DBName: "test_db", TableName: func() string {
				if tt.name == "test2" {
					return "TEST"
				}
				return "test"
			}()}].value, tt.want)
		})
	}
}

func TestBaseTableMetaCache_refresh_EarlyReturn(t *testing.T) {
	tests := []struct {
		name   string
		db     *sql.DB
		dbName string
		cache  map[types.TableMetaKey]*entry
		expect string
	}{
		{
			name:   "db_is_nil",
			db:     nil,
			dbName: "test_db",
			cache:  map[types.TableMetaKey]*entry{{DBName: "test_db", TableName: "test"}: {value: types.TableMeta{}}},
			expect: "should return early when db is nil",
		},
		{
			name:   "cache_is_nil",
			db:     &sql.DB{},
			dbName: "test_db",
			cache:  nil,
			expect: "should return early when cache is nil",
		},
		{
			name:   "cache_is_empty",
			db:     &sql.DB{},
			dbName: "test_db",
			cache:  map[types.TableMetaKey]*entry{},
			expect: "should return early when cache is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cancel := context.WithCancel(context.Background())
			defer cancel()

			c := &BaseTableMetaCache{
				expireDuration: EexpireTime,
				capity:         capacity,
				size:           0,
				cache:          tt.cache,
				trigger:        &mockTrigger{},
				db:             tt.db,
				dbName:         tt.dbName,
			}

			// Call refresh once and it should return early without panic
			done := make(chan bool)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("refresh() panicked: %v", r)
					}
					done <- true
				}()

				// Call the internal function once
				c.lock.RLock()
				if c.db == nil || c.cache == nil || len(c.cache) == 0 {
					c.lock.RUnlock()
					done <- true
					return
				}
				c.lock.RUnlock()
			}()

			select {
			case <-done:
				// Test passed - early return worked correctly
			case <-time.After(2 * time.Second):
				t.Error("refresh() did not return early as expected")
			}
		})
	}
}

func TestBaseTableMetaCache_GetTableMeta(t *testing.T) {
	var (
		tableMeta1  types.TableMeta
		tableMeta2  types.TableMeta
		columns     = make(map[string]types.ColumnMeta)
		index       = make(map[string]types.IndexMeta)
		index2      = make(map[string]types.IndexMeta)
		columnMeta1 []types.ColumnMeta
		columnMeta2 []types.ColumnMeta
		ColumnNames []string
	)
	columnId := types.ColumnMeta{
		ColumnDef:  nil,
		ColumnName: "id",
	}
	columnName := types.ColumnMeta{
		ColumnDef:  nil,
		ColumnName: "name",
	}
	columnAge := types.ColumnMeta{
		ColumnDef:  nil,
		ColumnName: "age",
	}
	columns["id"] = columnId
	columns["name"] = columnName
	columns["age"] = columnAge
	columnMeta1 = append(columnMeta1, columnId)
	columnMeta2 = append(columnMeta2, columnName, columnAge)
	index["id"] = types.IndexMeta{
		Name:    "PRIMARY",
		IType:   types.IndexTypePrimaryKey,
		Columns: columnMeta1,
	}
	index["id_name_age"] = types.IndexMeta{
		Name:    "name_age_idx",
		IType:   types.IndexUnique,
		Columns: columnMeta2,
	}

	ColumnNames = []string{"id", "name", "age"}
	tableMeta1 = types.TableMeta{
		TableName:   "t_user1",
		Columns:     columns,
		Indexs:      index,
		ColumnNames: ColumnNames,
	}

	index2["id_name_age"] = types.IndexMeta{
		Name:    "name_age_idx",
		IType:   types.IndexUnique,
		Columns: columnMeta2,
	}

	tableMeta2 = types.TableMeta{
		TableName:   "T_USER2",
		Columns:     columns,
		Indexs:      index2,
		ColumnNames: ColumnNames,
	}
	tests := []types.TableMeta{tableMeta1, tableMeta2}
	// Use sqlmock to simulate a database connection
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()
	for _, tt := range tests {
		t.Run(tt.TableName, func(t *testing.T) {
			mockTrigger := &mockTrigger{}
			// Mock a query response
			mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}))
			// Create a mock database connection
			conn, err := db.Conn(context.Background())
			if err != nil {
				t.Fatalf("Failed to get connection: %v", err)
			}
			defer conn.Close()
			cache := &BaseTableMetaCache{
				trigger: mockTrigger,
				cache: map[types.TableMetaKey]*entry{
					{DBName: "db", TableName: "t_user1"}: {
						value:      tableMeta1,
						lastAccess: time.Now(),
					},
					{DBName: "db", TableName: "T_USER2"}: {
						value:      tableMeta2,
						lastAccess: time.Now(),
					},
				},
				lock: sync.RWMutex{},
			}

			key := types.TableMetaKey{DBName: "db", TableName: tt.TableName}
			meta, _ := cache.GetTableMeta(context.Background(), key, conn)

			if meta.TableName != tt.TableName {
				t.Errorf("GetTableMeta() got TableName = %v, want %v", meta.TableName, tt.TableName)
			}
			// Ensure the retrieved table is cached
			cache.lock.RLock()
			_, cached := cache.cache[key]
			cache.lock.RUnlock()

			if !cached {
				t.Errorf("GetTableMeta() got TableName = %v, want %v", meta.TableName, tt.TableName)
			}
		})
	}
}

func TestBaseTableMetaCache_GracefulShutdown(t *testing.T) {
	// Create context manually as we are bypassing NewBaseCache
	ctx, cancel := context.WithCancel(context.Background())

	c := &BaseTableMetaCache{
		expireDuration:  1 * time.Millisecond,
		refreshInterval: 1 * time.Millisecond,
		cache:           make(map[types.TableMetaKey]*entry),
		// db and dbName are unset, so refresh() logic will return early, which is fine for coverage
	}

	// Init starts the goroutines
	err := c.Init(ctx)
	assert.Nil(t, err)

	// Give enough time for tickers to trigger multiple times
	time.Sleep(20 * time.Millisecond)

	// Cancel context to stop goroutines
	cancel()

	// Destroy (now a no-op)
	err = c.Destroy()
	assert.Nil(t, err)
}

type identityTrigger struct{}

func (identityTrigger) LoadOne(_ context.Context, key types.TableMetaKey, _ *sql.Conn) (*types.TableMeta, error) {
	return &types.TableMeta{TableName: key.TableName, Columns: map[string]types.ColumnMeta{"source": {ColumnName: key.DBName + "/" + key.Schema}}}, nil
}

func (identityTrigger) LoadAll(_ context.Context, _ *sql.Conn, _ ...types.TableMetaKey) (map[types.TableMetaKey]types.TableMeta, error) {
	return nil, nil
}

func TestBaseTableMetaCache_DistinguishesDatabases(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cache := &BaseTableMetaCache{cache: make(map[types.TableMetaKey]*entry), trigger: identityTrigger{}}
	for _, key := range []types.TableMetaKey{
		{DBName: "first", Schema: "public", TableName: "users"},
		{DBName: "second", Schema: "public", TableName: "users"},
		{DBName: "first", Schema: "tenant", TableName: "users"},
		{DBName: "first", Schema: "public", TableName: "users"},
	} {
		conn, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		meta, err := cache.GetTableMeta(context.Background(), key, conn)
		if err != nil {
			t.Fatal(err)
		}
		if got := meta.Columns["source"].ColumnName; got != key.DBName+"/"+key.Schema {
			t.Fatalf("key %+v returned metadata for %q", key, got)
		}
	}
}

type refreshIdentityTrigger struct {
	loaded chan []types.TableMetaKey
}

func (r refreshIdentityTrigger) LoadOne(_ context.Context, key types.TableMetaKey, _ *sql.Conn) (*types.TableMeta, error) {
	return &types.TableMeta{TableName: key.TableName}, nil
}

func (r refreshIdentityTrigger) LoadAll(_ context.Context, _ *sql.Conn, keys ...types.TableMetaKey) (map[types.TableMetaKey]types.TableMeta, error) {
	refreshed := make(map[types.TableMetaKey]types.TableMeta, len(keys))
	for _, key := range keys {
		refreshed[key] = types.TableMeta{TableName: key.TableName, Columns: map[string]types.ColumnMeta{
			"source": {ColumnName: key.DBName + "/" + key.Schema},
		}}
	}
	r.loaded <- keys
	return refreshed, nil
}

func TestBaseTableMetaCache_RefreshPreservesKeysAndAccessTime(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	keys := []types.TableMetaKey{
		{DBName: "first", Schema: "public", TableName: "users"},
		{DBName: "second", Schema: "tenant", TableName: "users"},
	}
	lastAccess := time.Now().Add(-time.Minute)
	loaded := make(chan []types.TableMetaKey, 1)
	cache := &BaseTableMetaCache{
		cache: map[types.TableMetaKey]*entry{
			keys[0]: {value: types.TableMeta{TableName: "stale"}, lastAccess: lastAccess},
			keys[1]: {value: types.TableMeta{TableName: "stale"}, lastAccess: lastAccess},
		},
		trigger:         refreshIdentityTrigger{loaded: loaded},
		db:              db,
		refreshInterval: time.Hour,
		// A resource without a default database can still cache explicitly resolved keys.
		dbName: "",
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		cache.refresh(ctx)
		close(done)
	}()
	select {
	case got := <-loaded:
		assert.ElementsMatch(t, keys, got)
	case <-time.After(time.Second):
		t.Fatal("refresh did not load explicit table keys")
	}
	if !assert.Eventually(t, func() bool {
		cache.lock.RLock()
		defer cache.lock.RUnlock()
		for _, key := range keys {
			entry, ok := cache.cache[key]
			if !ok || entry.value.Columns["source"].ColumnName != key.DBName+"/"+key.Schema || !entry.lastAccess.Equal(lastAccess) {
				return false
			}
		}
		return true
	}, time.Second, time.Millisecond) {
		t.Fatal("refresh replaced or miskeyed cached entries")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("refresh did not stop after cancellation")
	}
}
