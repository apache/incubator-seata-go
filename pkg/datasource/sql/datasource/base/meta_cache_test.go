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
	"github.com/agiledragon/gomonkey/v2"
	"seata.apache.org/seata-go/testdata"
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
		cfg            interface{} // 保留当前分支的interface{}以支持多数据库配置
	}
	type args struct {
		ctx context.Context
	}

	// 公共配置
	mysqlCfg := &mysql.Config{DBName: "test_mysql_db"}
	postgresDSN := "host=localhost port=5432 user=test dbname=test_postgres_db password=test"
	expireTime := 10 * time.Millisecond // 统一过期时间变量

	// 创建sqlmock实例（当前分支的模拟方式，master分支可复用）
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
		name    string
		fields  func(ctx context.Context, cancel context.CancelFunc) fields
		args    args
		want    types.TableMeta // 保留master分支的预期结果字段
		wantErr bool
	}{
		// 1. 当前分支的MySQL测试场景
		{
			name: "mysql_config_refresh",
			fields: func(ctx context.Context, cancel context.CancelFunc) fields {
				return fields{
					lock:           sync.RWMutex{},
					capity:         capacity,
					expireDuration: expireTime,
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
			want: testdata.MockWantTypesMeta("TEST_MYSQL_TABLE"), // 复用master的预期结果生成方法
		},
		// 2. 当前分支的PostgreSQL测试场景
		{
			name: "postgres_config_refresh",
			fields: func(ctx context.Context, cancel context.CancelFunc) fields {
				return fields{
					lock:           sync.RWMutex{},
					capity:         capacity,
					expireDuration: expireTime,
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
			want: testdata.MockWantTypesMeta("TEST_POSTGRES_TABLE"),
		},
		// 3. master分支的test1（表名小写）
		{
			name: "test1_lowercase_table",
			fields: func(ctx context.Context, cancel context.CancelFunc) fields {
				return fields{
					lock:           sync.RWMutex{},
					capity:         capacity,
					size:           0,
					expireDuration: expireTime,
					cache: map[string]*entry{
						"test": {
							value:      types.TableMeta{},
							lastAccess: time.Now(),
						},
					},
					cancel:  cancel,
					trigger: &mockTrigger{},
					cfg:     mysqlCfg, // 复用MySQL配置
					db:      mockDB,   // 复用sqlmock
				}
			},
			args: args{},
			want: testdata.MockWantTypesMeta("test"),
		},
		// 4. master分支的test2（表名大写）
		{
			name: "test2_uppercase_table",
			fields: func(ctx context.Context, cancel context.CancelFunc) fields {
				return fields{
					lock:           sync.RWMutex{},
					capity:         capacity,
					size:           0,
					expireDuration: expireTime,
					cache: map[string]*entry{
						"TEST": {
							value:      types.TableMeta{},
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
			want: testdata.MockWantTypesMeta("TEST"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 统一使用带超时的context（当前分支的机制，兼容master的取消逻辑）
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer timeoutCancel()

			// 初始化测试字段
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

			// 复用master分支的gomonkey桩（处理数据库连接和加载逻辑）
			connStub := gomonkey.ApplyMethodFunc(fields.db, "Conn",
				func(_ context.Context) (*sql.Conn, error) {
					return &sql.Conn{}, nil
				})
			defer connStub.Reset()

			loadAllStub := gomonkey.ApplyMethodFunc(fields.trigger, "LoadAll",
				func(_ context.Context, _ string, _ *sql.Conn, _ ...string) ([]types.TableMeta, error) {
					return []types.TableMeta{tt.want}, nil
				})
			defer loadAllStub.Reset()

			// 异步执行refresh并等待完成（当前分支的并发控制）
			refreshDone := make(chan struct{})
			go func() {
				defer close(refreshDone)
				c.refresh(timeoutCtx)
			}()

			// 等待刷新逻辑执行（结合当前分支的短等待和master的长等待，取折中值）
			time.Sleep(20 * time.Millisecond)
			timeoutCancel() // 触发取消信号

			// 校验是否正常退出（当前分支的退出校验）
			select {
			case <-refreshDone:
			case <-time.After(100 * time.Millisecond):
				t.Fatal("refresh method did not respond to cancel signal, timed out")
			}

			// 校验缓存结果（合并双方的校验逻辑：存在性+值正确性）
			c.lock.RLock()
			defer c.lock.RUnlock()

			// 确定当前测试用例的表名（适配不同场景的表名）
			tableName := ""
			switch tt.name {
			case "mysql_config_refresh":
				tableName = "TEST_MYSQL_TABLE"
			case "postgres_config_refresh":
				tableName = "TEST_POSTGRES_TABLE"
			case "test1_lowercase_table":
				tableName = "test"
			case "test2_uppercase_table":
				tableName = "TEST"
			}

			// 1. 校验表是否在缓存中（当前分支的逻辑）
			assert.Contains(t, c.cache, tableName, "table %s not found in cache", tableName)
			// 2. 校验列名非空（当前分支的逻辑）
			assert.NotEmpty(t, c.cache[tableName].value.ColumnNames, "table %s metadata not refreshed", tableName)
			// 3. 校验缓存值是否与预期一致（master分支的逻辑）
			assert.Equal(t, c.cache[tableName].value, tt.want, "table %s metadata mismatch", tableName)
		})
	}
}

// TestBaseTableMetaCache_GetTableMeta
func TestBaseTableMetaCache_GetTableMeta(t *testing.T) {
	// 扩展表元数据创建函数，整合master分支的详细列和索引定义
	createTestTableMeta := func(tableName string, hasPrimaryKey bool) types.TableMeta {
		// 定义列元数据（复用master分支的详细定义）
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
		columns := map[string]types.ColumnMeta{
			"id":   columnId,
			"name": columnName,
			"age":  columnAge,
		}

		// 定义索引元数据（整合master分支的多索引场景）
		indexes := make(map[string]types.IndexMeta)
		columnMeta1 := []types.ColumnMeta{columnId}       // 主键索引列
		columnMeta2 := []types.ColumnMeta{columnName, columnAge} // 联合索引列

		if hasPrimaryKey {
			// 主键索引（master分支的test1场景）
			indexes["id"] = types.IndexMeta{
				Name:    "PRIMARY",
				IType:   types.IndexTypePrimaryKey,
				Columns: columnMeta1,
			}
		}
		// 唯一联合索引（master分支的通用场景）
		indexes["id_name_age"] = types.IndexMeta{
			Name:    "name_age_idx",
			IType:   types.IndexUnique,
			Columns: columnMeta2,
		}

		return types.TableMeta{
			TableName:   tableName,
			Columns:     columns,
			Indexs:      indexes,
			ColumnNames: []string{"id", "name", "age"},
		}
	}

	// 测试用例：整合master的大小写表名与不同索引场景
	tests := []types.TableMeta{
		createTestTableMeta("t_user1", true),  // 小写表名+含主键索引（对应master的test1）
		createTestTableMeta("T_USER2", false), // 大写表名+无主键索引（对应master的test2）
	}

	// 创建sqlmock实例（复用双方的模拟数据库方式）
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 多数据库配置（保留当前分支的扩展能力）
	configs := []struct {
		name string
		cfg  interface{}
	}{
		{"mysql", &mysql.Config{DBName: "test_db"}},
		{"postgres", "host=localhost dbname=test_db user=test"},
	}

	// 外层循环：测试不同数据库配置
	for _, cfg := range configs {
		t.Run("config_"+cfg.name, func(t *testing.T) {
			// 内层循环：测试不同表元数据
			for _, tt := range tests {
				t.Run(tt.TableName, func(t *testing.T) {
					// 初始化mock触发器和数据库连接
					mockTrigger := &mockTrigger{}
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}))

					conn, err := db.Conn(context.Background())
					if err != nil {
						t.Fatalf("Failed to get connection: %v", err)
					}
					defer conn.Close()

					// 初始化缓存（整合双方的缓存数据）
					cache := &BaseTableMetaCache{
						trigger: mockTrigger,
						cache: map[string]*entry{
							strings.ToUpper("t_user1"): { // 统一缓存键为大写（当前分支的处理逻辑）
								value:      createTestTableMeta("t_user1", true),
								lastAccess: time.Now(),
							},
							strings.ToUpper("T_USER2"): {
								value:      createTestTableMeta("T_USER2", false),
								lastAccess: time.Now(),
							},
						},
						lock: sync.RWMutex{},
						cfg:  cfg.cfg, // 注入当前数据库配置
					}

					// 执行测试方法
					meta, err := cache.GetTableMeta(context.Background(), "test_db", tt.TableName, conn)

					// 断言：无错误（当前分支的assert风格）
					assert.NoError(t, err, "GetTableMeta returned unexpected error")
					// 断言：表名匹配（整合双方的校验点）
					assert.Equal(t, tt.TableName, meta.TableName, "Table name mismatch")
					// 断言：索引数量匹配（补充master分支的元数据细节校验）
					assert.Equal(t, len(tt.Indexs), len(meta.Indexs), "Index count mismatch")

					// 校验缓存是否生效（统一使用大写键校验，当前分支的逻辑）
					cache.lock.RLock()
					entry, cached := cache.cache[strings.ToUpper(tt.TableName)]
					cache.lock.RUnlock()

					assert.True(t, cached, "Table %s not cached", tt.TableName)
					assert.Equal(t, tt.TableName, entry.value.TableName, "Cached table name mismatch")
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
