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
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/testdata"
)

var (
	capacity      int32 = 1024
	EexpireTime         = 15 * time.Minute
	tableMetaOnce sync.Once
)

type mockTrigger struct {
}

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
	return nil, nil
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
		cfg            *mysql.Config
	}
	type args struct {
		ctx context.Context
	}
	ctx, cancel := context.WithCancel(context.Background())
	tests := []struct {
		name   string
		fields fields
		args   args
		want   types.TableMeta
	}{
		{name: "test-1",
			fields: fields{
				lock:           sync.RWMutex{},
				capity:         capacity,
				size:           0,
				expireDuration: EexpireTime,
				cache: map[string]*entry{
					"TEST": {
						value:      types.TableMeta{},
						lastAccess: time.Now(),
					},
				},
				cancel:  cancel,
				trigger: &mockTrigger{},
				cfg:     &mysql.Config{},
				db:      &sql.DB{},
			}, args: args{ctx: ctx},
			want: testdata.MockWantTypesMeta("test")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			connStub := gomonkey.ApplyMethodFunc(tt.fields.db, "Conn",
				func(_ context.Context) (*sql.Conn, error) {
					return &sql.Conn{}, nil
				})

			defer connStub.Reset()

			loadAllStub := gomonkey.ApplyMethodFunc(tt.fields.trigger, "LoadAll",
				func(_ context.Context, _ string, _ *sql.Conn, _ ...string) ([]types.TableMeta, error) {
					return []types.TableMeta{tt.want}, nil
				})

			defer loadAllStub.Reset()

			c := &BaseTableMetaCache{
				lock:           tt.fields.lock,
				expireDuration: tt.fields.expireDuration,
				capity:         tt.fields.capity,
				size:           tt.fields.size,
				cache:          tt.fields.cache,
				cancel:         tt.fields.cancel,
				trigger:        tt.fields.trigger,
				db:             tt.fields.db,
				cfg:            tt.fields.cfg,
			}
			go c.refresh(tt.args.ctx)
			time.Sleep(time.Second * 3)

			assert.Equal(t, c.cache["TEST"].value, tt.want)
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
		TableName:   "T_USER1",
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
				cache: map[string]*entry{
					"T_USER": {
						value:      tableMeta2,
						lastAccess: time.Now(),
					},
					"T_USER1": {
						value:      tableMeta1,
						lastAccess: time.Now(),
					},
				},
				lock: sync.RWMutex{},
			}

			meta, _ := cache.GetTableMeta(context.Background(), "db", tt.TableName, conn)

			if meta.TableName != tt.TableName {
				t.Errorf("GetTableMeta() got TableName = %v, want %v", meta.TableName, tt.TableName)
			}
			// Ensure the retrieved table is cached
			cache.lock.RLock()
			_, cached := cache.cache[tt.TableName]
			cache.lock.RUnlock()

			if !cached {
				t.Errorf("GetTableMeta() got TableName = %v, want %v", meta.TableName, tt.TableName)
			}
		})
	}
}
