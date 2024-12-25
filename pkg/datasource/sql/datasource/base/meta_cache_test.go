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

	"seata.apache.org/seata-go/pkg/datasource/sql/types"

	"testing"
	"time"
)

type mockTrigger struct {
	data []types.TableMeta
	err  error
}

func (mt *mockTrigger) LoadAll() ([]types.TableMeta, error) {
	return mt.data, mt.err
}

func (mt *mockTrigger) LoadOne(ctx context.Context, dbName string, table string, conn *sql.Conn) (*types.TableMeta, error) {
	if mt.err != nil {
		return nil, mt.err
	}
	for _, meta := range mt.data {
		if meta.TableName == table {
			return &meta, nil
		}
	}

	return nil, nil
}

func TestBaseTableMetaCache_refresh(t *testing.T) {
	tests := []struct {
		name    string
		trigger *mockTrigger
		cache   map[string]*entry
		wantErr bool
	}{
		{
			name: "正常情况",
			trigger: &mockTrigger{
				data: []types.TableMeta{
					{TableName: "test1"},
					{TableName: "test2"},
				},
				err: nil,
			},
			cache:   make(map[string]*entry),
			wantErr: false,
		},
		{
			name: "缓存中已存在",
			trigger: &mockTrigger{
				data: []types.TableMeta{
					{TableName: "test1"},
				},
				err: nil,
			},
			cache: map[string]*entry{
				"TEST1": {value: types.TableMeta{TableName: "test1"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &BaseTableMetaCache{
				trigger: tt.trigger,
				cache:   tt.cache,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			go c.refresh(ctx)

			time.Sleep(1 * time.Second)

			c.lock.Lock()
			if len(c.cache) != len(tt.trigger.data) {
				t.Errorf("cache size = %v, want %v", len(c.cache), len(tt.trigger.data))
			}
			c.lock.Unlock()
		})
	}
}

func TestBaseTableMetaCache_GetTableMeta(t *testing.T) {
	var (
		tableMeta1 types.TableMeta
		// one index table
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
		TableName:   "t_user",
		Columns:     columns,
		Indexs:      index2,
		ColumnNames: ColumnNames,
	}
	tests := []types.TableMeta{tableMeta1, tableMeta2}
	for _, tt := range tests {
		t.Run(tt.TableName, func(t *testing.T) {
			mockTrigger := &mockTrigger{
				data: tests,
				err:  nil,
			}

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
			mockConn := &sql.Conn{}

			meta, _ := cache.GetTableMeta(context.Background(), "db", tt.TableName, mockConn)

			if meta.TableName != tt.TableName {
				t.Errorf("GetTableMeta() got TableName = %v, want %v", meta.TableName, tt.TableName)
			}

			cache.lock.RLock()
			_, cached := cache.cache[tt.TableName]
			cache.lock.RUnlock()

			if !cached {
				t.Errorf("GetTableMeta() got TableName = %v, want %v", meta.TableName, tt.TableName)
			}
		})
	}
}
