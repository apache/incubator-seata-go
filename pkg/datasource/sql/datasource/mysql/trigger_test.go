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

package mysql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/testdata"

	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
)

func initMockIndexMeta() []types.IndexMeta {
	return []types.IndexMeta{
		{
			IType:      types.IndexTypePrimaryKey,
			ColumnName: "id",
			Columns: []types.ColumnMeta{
				{
					ColumnName:   "id",
					DatabaseType: types.GetSqlDataType("BIGINT"),
				},
			},
		},
	}
}

func initMockColumnMeta() []types.ColumnMeta {
	return []types.ColumnMeta{
		{
			ColumnName: "id",
		},
		{
			ColumnName: "name",
		},
	}
}

func initGetIndexesStub(m *mysqlTrigger, indexMeta []types.IndexMeta) *gomonkey.Patches {
	getIndexesStub := gomonkey.ApplyPrivateMethod(m, "getIndexes",
		func(_ *mysqlTrigger, ctx context.Context, dbName string, tableName string, conn *sql.Conn) ([]types.IndexMeta, error) {
			return indexMeta, nil
		})
	return getIndexesStub
}

func initGetColumnMetasStub(m *mysqlTrigger, columnMeta []types.ColumnMeta) *gomonkey.Patches {
	getColumnMetasStub := gomonkey.ApplyPrivateMethod(m, "getColumnMetas",
		func(_ *mysqlTrigger, ctx context.Context, dbName string, table string, conn *sql.Conn) ([]types.ColumnMeta, error) {
			return columnMeta, nil
		})
	return getColumnMetasStub
}

func Test_mysqlTrigger_LoadOne(t *testing.T) {
	wantTableMeta := testdata.MockWantTypesMeta("test")
	type args struct {
		ctx       context.Context
		dbName    string
		tableName string
		conn      *sql.Conn
	}
	tests := []struct {
		name          string
		args          args
		columnMeta    []types.ColumnMeta
		indexMeta     []types.IndexMeta
		wantTableMeta *types.TableMeta
	}{
		{
			name:          "1",
			args:          args{ctx: context.Background(), dbName: "dbName", tableName: "test", conn: nil},
			indexMeta:     initMockIndexMeta(),
			columnMeta:    initMockColumnMeta(),
			wantTableMeta: &wantTableMeta,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mysqlTrigger{}

			getColumnMetasStub := initGetColumnMetasStub(m, tt.columnMeta)
			defer getColumnMetasStub.Reset()

			getIndexesStub := initGetIndexesStub(m, tt.indexMeta)
			defer getIndexesStub.Reset()

			got, err := m.LoadOne(tt.args.ctx, tt.args.dbName, tt.args.tableName, tt.args.conn)
			if err != nil {
				t.Errorf("LoadOne() error = %v", err)
				return
			}

			assert.Equal(t, tt.wantTableMeta, got)
		})
	}
}

func initMockResourceManager(branchType branch.BranchType, ctrl *gomock.Controller) *mock.MockDataSourceManager {
	mockResourceMgr := mock.NewMockDataSourceManager(ctrl)
	mockResourceMgr.SetBranchType(branchType)
	mockResourceMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).AnyTimes().Return(int64(0), nil)
	rm.GetRmCacheInstance().RegisterResourceManager(mockResourceMgr)
	mockResourceMgr.EXPECT().RegisterResource(gomock.Any()).AnyTimes().Return(nil)
	mockResourceMgr.EXPECT().CreateTableMetaCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	return mockResourceMgr
}

func Test_mysqlTrigger_LoadAll(t *testing.T) {
	sql.Register("seata-at-mysql", &mock.MockTestDriver{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	conn := sql.Conn{}
	type args struct {
		ctx    context.Context
		dbName string
		conn   *sql.Conn
		tables []string
	}
	tests := []struct {
		name       string
		args       args
		columnMeta []types.ColumnMeta
		indexMeta  []types.IndexMeta
		want       []types.TableMeta
	}{
		{
			name: "test-01",
			args: args{
				ctx:    nil,
				dbName: "dbName",
				conn:   &conn,
				tables: []string{
					"test_01",
					"test_02",
				},
			},
			indexMeta:  initMockIndexMeta(),
			columnMeta: initMockColumnMeta(),
			want:       []types.TableMeta{testdata.MockWantTypesMeta("test_01"), testdata.MockWantTypesMeta("test_02")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mysqlTrigger{}

			getColumnMetasStub := initGetColumnMetasStub(m, tt.columnMeta)
			defer getColumnMetasStub.Reset()

			getIndexesStub := initGetIndexesStub(m, tt.indexMeta)
			defer getIndexesStub.Reset()

			got, err := m.LoadAll(tt.args.ctx, tt.args.dbName, tt.args.conn, tt.args.tables...)
			if err != nil {
				t.Errorf("LoadAll() error = %v", err)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
