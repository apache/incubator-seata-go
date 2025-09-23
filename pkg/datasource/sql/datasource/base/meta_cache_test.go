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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	capacity      int32 = 1024
	expireTime          = 15 * time.Minute
	tableMetaOnce sync.Once
)

type mockTrigger struct{}

// LoadOne simulates loading table metadata, including id, name, and age columns.
func (m *mockTrigger) LoadOne(ctx context.Context, dbName string, table string, conn *sql.Conn) (*types.TableMeta, error) {

	return &types.TableMeta{
		TableName: table,
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

func (m *mockTrigger) LoadAll(ctx context.Context, dbName string, conn *sql.Conn, tables ...string) ([]types.TableMeta, error) {
	tableMetas := make([]types.TableMeta, 0, len(tables))
	for _, table := range tables {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		tableMetas = append(tableMetas, types.TableMeta{
			TableName: table,
			Columns: map[string]types.ColumnMeta{
				"id":   {ColumnName: "id"},
				"name": {ColumnName: "name"},
			},
			ColumnNames: []string{"id", "name"},
		})
	}
	return tableMetas, nil
}

func TestBaseTableMetaCache_refresh(t *testing.T) {
	type fields struct {
		lock           sync.RWMutex
		expireDuration time.Duration
		capity         int32
		size           int32
		cache          map[string]*entry
		cancel         context.CancelFunc
		trigger        trigger
		db             *sql.DB
		cfg            interface{}
	}
	type args struct {
		ctx context.Context
	}

	mysqlCfg := &mysql.Config{DBName: "test_mysql_db"}
	postgresDSN := "host=localhost port=5432 user=test dbname=test_postgres_db password=test"

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer func() {
		mockDB.Close()
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %s", err)
		}
	}()

	tests := []struct {
		name   string
		fields func(ctx context.Context, cancel context.CancelFunc) fields
		args   args
	}{
		{
			name: "mysql_config_refresh",
			fields: func(ctx context.Context, cancel context.CancelFunc) fields {
				return fields{
					lock:           sync.RWMutex{},
					capity:         capacity,
					expireDuration: 10 * time.Millisecond,
					cache: map[string]*entry{
						"TEST_MYSQL_TABLE": {
							value:      types.TableMeta{TableName: "TEST_MYSQL_TABLE"},
							lastAccess: time.Now(),
						},
					},
					cancel:  cancel,
					trigger: &mockTrigger{},
					cfg:     mysqlCfg,
					db:      mockDB,
				}
			},
			args: args{},
		},
		{
			name: "postgres_config_refresh",
			fields: func(ctx context.Context, cancel context.CancelFunc) fields {
				return fields{
					lock:           sync.RWMutex{},
					capity:         capacity,
					expireDuration: 10 * time.Millisecond,
					cache: map[string]*entry{
						"TEST_POSTGRES_TABLE": {
							value:      types.TableMeta{TableName: "TEST_POSTGRES_TABLE"},
							lastAccess: time.Now(),
						},
					},
					cancel:  cancel,
					trigger: &mockTrigger{},
					cfg:     postgresDSN,
					db:      mockDB,
				}
			},
			args: args{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer timeoutCancel()

			fields := tt.fields(timeoutCtx, timeoutCancel)
			c := &BaseTableMetaCache{
				lock:           fields.lock,
				expireDuration: fields.expireDuration,
				capity:         fields.capity,
				size:           fields.size,
				cache:          fields.cache,
				cancel:         fields.cancel,
				trigger:        fields.trigger,
				db:             fields.db,
				cfg:            fields.cfg,
			}

			refreshDone := make(chan struct{})

			go func() {
				defer close(refreshDone)
				c.refresh(timeoutCtx)
			}()

			time.Sleep(20 * time.Millisecond)
			timeoutCancel()

			select {
			case <-refreshDone:
			case <-time.After(100 * time.Millisecond):
				t.Fatal("The refresh method did not respond to the cancel signal, and timed out on exit")
			}

			c.lock.RLock()
			defer c.lock.RUnlock()
			for tableName := range fields.cache {
				assert.Contains(t, c.cache, tableName, "Table %s not found in cache", tableName)
				assert.NotEmpty(t, c.cache[tableName].value.ColumnNames, "Table %s metadata has not been refreshed", tableName)
			}
		})
	}
}

// TestBaseTableMetaCache_GetTableMeta
func TestBaseTableMetaCache_GetTableMeta(t *testing.T) {

	createTestTableMeta := func(tableName string) types.TableMeta {
		columns := map[string]types.ColumnMeta{
			"id":   {ColumnName: "id"},
			"name": {ColumnName: "name"},
			"age":  {ColumnName: "age"},
		}
		indexes := map[string]types.IndexMeta{
			"id": {
				Name:    "PRIMARY",
				IType:   types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
			},
		}
		return types.TableMeta{
			TableName:   tableName,
			Columns:     columns,
			Indexs:      indexes,
			ColumnNames: []string{"id", "name", "age"},
		}
	}

	tableMeta1 := createTestTableMeta("T_USER1")
	tableMeta2 := createTestTableMeta("T_USER2")
	tests := []types.TableMeta{tableMeta1, tableMeta2}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	configs := []struct {
		name string
		cfg  interface{}
	}{
		{"mysql", &mysql.Config{DBName: "test_db"}},
		{"postgres", "host=localhost dbname=test_db user=test"},
	}

	for _, cfg := range configs {
		t.Run("config_"+cfg.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.TableName, func(t *testing.T) {
					mockTrigger := &mockTrigger{}
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}))

					conn, err := db.Conn(context.Background())
					if err != nil {
						t.Fatalf("Failed to get connection: %v", err)
					}
					defer conn.Close()

					cache := &BaseTableMetaCache{
						trigger: mockTrigger,
						cache: map[string]*entry{
							tableMeta1.TableName: {
								value:      tableMeta1,
								lastAccess: time.Now(),
							},
						},
						lock: sync.RWMutex{},
						cfg:  cfg.cfg,
					}

					meta, err := cache.GetTableMeta(context.Background(), "test_db", tt.TableName, conn)
					assert.NoError(t, err)
					assert.Equal(t, tt.TableName, meta.TableName)

					cache.lock.RLock()
					entry, cached := cache.cache[strings.ToUpper(tt.TableName)]
					cache.lock.RUnlock()

					assert.True(t, cached)
					assert.Equal(t, tt.TableName, entry.value.TableName)
				})
			}
		})
	}
}

// TestBaseTableMetaCache_getDBName
func TestBaseTableMetaCache_getDBName(t *testing.T) {
	tests := []struct {
		name    string
		cfg     interface{}
		wantDB  string
		wantErr bool
	}{
		{
			name:    "mysql_config",
			cfg:     &mysql.Config{DBName: "mysql_db"},
			wantDB:  "mysql_db",
			wantErr: false,
		},
		{
			name:    "postgres_dsn_full",
			cfg:     "host=localhost port=5432 user=test dbname=postgres_db password=secret",
			wantDB:  "postgres_db",
			wantErr: false,
		},
		{
			name:    "postgres_dsn_simple",
			cfg:     "dbname=simple_db",
			wantDB:  "simple_db",
			wantErr: false,
		},
		{
			name:    "unsupported_type",
			cfg:     12345,
			wantDB:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &BaseTableMetaCache{
				cfg: tt.cfg,
			}
			dbName, err := cache.getDBName()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantDB, dbName)
			}
		})
	}
}

// TestBaseTableMetaCache_scanExpire
func TestBaseTableMetaCache_scanExpire_Real(t *testing.T) {
	expireDuration := 100 * time.Millisecond

	cacheData := map[string]*entry{
		"EXPIRED_TABLE": {
			value:      types.TableMeta{TableName: "EXPIRED_TABLE"},
			lastAccess: time.Now().Add(-500 * time.Millisecond),
		},
		"VALID_TABLE": {
			value:      types.TableMeta{TableName: "VALID_TABLE"},
			lastAccess: time.Now(),
		},
	}

	c := &BaseTableMetaCache{
		lock:           sync.RWMutex{},
		expireDuration: expireDuration,
		cache:          cacheData,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})

	go func() {
		c.scanExpire(ctx)
		close(done)
	}()

	time.Sleep(expireDuration/2 + 10*time.Millisecond)

	c.lock.Lock()
	c.cache["VALID_TABLE"].lastAccess = time.Now()
	c.lock.Unlock()

	time.Sleep(expireDuration + 10*time.Millisecond)

	cancel()
	<-done

	c.lock.RLock()
	defer c.lock.RUnlock()

	assert.NotContains(t, c.cache, "EXPIRED_TABLE", "overdue items should be cleared")
	assert.Contains(t, c.cache, "VALID_TABLE", "Items that have not expired should be retained")
}
