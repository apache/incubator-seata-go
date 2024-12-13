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

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	capacity      int32 = 1024
	EexpireTime         = 15 * time.Minute
	tableMetaOnce sync.Once
)

type mockTrigger struct {
}

func (m *mockTrigger) LoadOne(ctx context.Context, dbName string, table string, conn *sql.Conn) (*types.TableMeta, error) {
	return nil, nil
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
					"test": &entry{
						value:      types.TableMeta{},
						lastAccess: time.Now(),
					},
				},
				cancel:  cancel,
				trigger: &mockTrigger{},
				cfg:     &mysql.Config{},
				db:      &sql.DB{},
			}, args: args{ctx: ctx},
			want: types.TableMeta{
				TableName: "test",
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
					},
					"name": {
						ColumnName: "name",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"": {
						ColumnName: "id",
						IType:      types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{
								ColumnName:   "id",
								DatabaseType: types.GetSqlDataType("BIGINT"),
							},
							{
								ColumnName: "id",
							},
						},
					},
				},
				ColumnNames: []string{
					"id",
					"name",
				},
			}},
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

			assert.Equal(t, c.cache["test"].value, tt.want)
		})
	}
}
