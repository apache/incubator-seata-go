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

package builder

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

func TestBuildWhereConditionByPKs(t *testing.T) {
	builder := BasicUndoLogBuilder{}
	tests := []struct {
		name       string
		pkNameList []string
		rowSize    int
		maxInSize  int
		expectSQL  string
	}{
		{"test1", []string{"id", "name"}, 1, 1, "(`id`,`name`) IN ((?,?))"},
		{"test1", []string{"id", "name"}, 3, 2, "(`id`,`name`) IN ((?,?),(?,?)) OR (`id`,`name`) IN ((?,?))"},
		{"test1", []string{"id", "name"}, 3, 1, "(`id`,`name`) IN ((?,?)) OR (`id`,`name`) IN ((?,?)) OR (`id`,`name`) IN ((?,?))"},
		{"test1", []string{"id", "name"}, 4, 2, "(`id`,`name`) IN ((?,?),(?,?)) OR (`id`,`name`) IN ((?,?),(?,?))"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// todo add dbType param
			sql := builder.buildWhereConditionByPKs(test.pkNameList, test.rowSize, "", test.maxInSize)
			assert.Equal(t, test.expectSQL, sql)
		})
	}
}

func TestBuildLockKey(t *testing.T) {
	var builder BasicUndoLogBuilder

	columnID := types.ColumnMeta{
		ColumnName: "id",
	}
	columnUserId := types.ColumnMeta{
		ColumnName: "userId",
	}
	columnName := types.ColumnMeta{
		ColumnName: "name",
	}
	columnAge := types.ColumnMeta{
		ColumnName: "age",
	}
	columnNonExistent := types.ColumnMeta{
		ColumnName: "non_existent",
	}

	columnsTwoPk := []types.ColumnMeta{columnID, columnUserId}
	columnsThreePk := []types.ColumnMeta{columnID, columnUserId, columnAge}
	columnsMixPk := []types.ColumnMeta{columnName, columnAge}

	getColumnImage := func(columnName string, value interface{}) types.ColumnImage {
		return types.ColumnImage{KeyType: types.IndexTypePrimaryKey, ColumnName: columnName, Value: value}
	}

	tests := []struct {
		name     string
		metaData types.TableMeta
		records  types.RecordImage
		expected string
	}{
		{
			"Two Primary Keys",
			types.TableMeta{
				TableName: "test_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsTwoPk},
				},
			},
			types.RecordImage{
				TableName: "test_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "one")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "two")}},
				},
			},
			"TEST_NAME:1_one,2_two",
		},
		{
			"Three Primary Keys",
			types.TableMeta{
				TableName: "test2_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsThreePk},
				},
			},
			types.RecordImage{
				TableName: "test2_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "one"), getColumnImage("age", "11")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "two"), getColumnImage("age", "22")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 3), getColumnImage("userId", "three"), getColumnImage("age", "33")}},
				},
			},
			"TEST2_NAME:1_one_11,2_two_22,3_three_33",
		},
		{
			"Three Primary Keys",
			types.TableMeta{
				TableName: "test2_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsThreePk},
				},
			},
			types.RecordImage{
				TableName: "test2_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "one"), getColumnImage("age", "11")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "two"), getColumnImage("age", "22")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 3), getColumnImage("userId", "three"), getColumnImage("age", "33")}},
				},
			},
			"TEST2_NAME:1_one_11,2_two_22,3_three_33",
		},
		{
			name: "Single Primary Key",
			metaData: types.TableMeta{
				TableName: "single_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "single_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 100)}},
				},
			},
			expected: "SINGLE_KEY:100",
		},
		{
			name: "Mixed Type Keys",
			metaData: types.TableMeta{
				TableName: "mixed_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsMixPk},
				},
			},
			records: types.RecordImage{
				TableName: "mixed_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("name", "Alice"), getColumnImage("age", 25)}},
				},
			},
			expected: "MIXED_KEY:Alice_25",
		},
		{
			name: "Empty Records",
			metaData: types.TableMeta{
				TableName: "empty",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records:  types.RecordImage{TableName: "empty"},
			expected: "EMPTY:",
		},
		{
			name: "Special Characters",
			metaData: types.TableMeta{
				TableName: "special",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "special",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", "a,b_c")}},
				},
			},
			expected: "SPECIAL:a,b_c",
		},
		{
			name: "Non-existent Key Name",
			metaData: types.TableMeta{
				TableName: "error_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnNonExistent}},
				},
			},
			records: types.RecordImage{
				TableName: "error_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1)}},
				},
			},
			expected: "ERROR_KEY:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := builder.buildLockKey(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}

// Only database I/O is replaced. Parsing, image building, lock generation
// and after-image SQL all use production code.
type legacyBuilderTestConn struct {
	prepared bool
	query    string
	args     []driver.Value
}

func (c *legacyBuilderTestConn) Prepare(query string) (driver.Stmt, error) {
	c.prepared, c.query = true, query
	return &legacyBuilderTestStmt{conn: c}, nil
}
func (*legacyBuilderTestConn) Close() error { return nil }
func (*legacyBuilderTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected Begin")
}

type legacyBuilderTestStmt struct{ conn *legacyBuilderTestConn }

func (*legacyBuilderTestStmt) Close() error  { return nil }
func (*legacyBuilderTestStmt) NumInput() int { return -1 }
func (*legacyBuilderTestStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unexpected Exec")
}
func (s *legacyBuilderTestStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.conn.args = append([]driver.Value(nil), args...)
	return &legacyBuilderTestRows{}, nil
}

type legacyBuilderTestRows struct{ next int64 }

func (*legacyBuilderTestRows) Columns() []string { return []string{"id"} }
func (*legacyBuilderTestRows) Close() error      { return nil }
func (r *legacyBuilderTestRows) Next(dest []driver.Value) error {
	if r.next == 2 {
		return io.EOF
	}
	if len(dest) != 1 {
		return fmt.Errorf("unexpected destination length: %d", len(dest))
	}
	r.next++
	dest[0] = r.next
	return nil
}

func legacyBuilderTestMeta() types.TableMeta {
	id := types.ColumnMeta{ColumnName: "id", DatabaseTypeString: "BIGINT"}
	return types.TableMeta{
		TableName: "t_user", ColumnNames: []string{"id"},
		Columns: map[string]types.ColumnMeta{"id": id},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{id}},
		},
	}
}

func legacyBuilderTestContext(t *testing.T, query string, conn driver.Conn) *types.ExecContext {
	t.Helper()
	parsed, err := parser.DoParser(query)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return &types.ExecContext{
		Query: query, ParseContext: parsed, Conn: conn, DBType: types.DBTypeMySQL,
		TxCtx:       &types.TransactionContext{LockKeys: map[string]struct{}{}},
		MetaDataMap: map[string]types.TableMeta{"t_user": legacyBuilderTestMeta()},
	}
}

func testBuilderBeforeImageParameters(t *testing.T, builder undo.UndoLogBuilder, query string, values []driver.Value) {
	t.Helper()
	log.Init()
	for _, named := range []bool{true, false} {
		mode := "Values"
		if named {
			mode = "NamedValues"
		}
		t.Run(mode, func(t *testing.T) {
			conn := &legacyBuilderTestConn{}
			execCtx := legacyBuilderTestContext(t, query, conn)
			if len(execCtx.ParseContext.MultiStmt) > 0 {
				assert.Nil(t, execCtx.ParseContext.UpdateStmt)
				assert.Nil(t, execCtx.ParseContext.DeleteStmt)
			}
			if named {
				for i, value := range values {
					execCtx.NamedValues = append(execCtx.NamedValues,
						driver.NamedValue{Ordinal: i + 1, Value: value})
				}
			} else {
				execCtx.Values = values
			}
			images, err := builder.BeforeImage(context.Background(), execCtx)
			if !assert.NoError(t, err) {
				return
			}
			if assert.Len(t, images, 1) {
				assert.Equal(t, "t_user", images[0].TableName)
				assert.Len(t, images[0].Rows, 2)
			}
			assert.True(t, conn.prepared)
			assert.Equal(t, []driver.Value{int64(1), int64(2)}, conn.args)
		})
	}
}

func testBuilderLockPrimaryKeys(t *testing.T, builder undo.UndoLogBuilder, query string) {
	t.Helper()
	log.Init()
	conn := &legacyBuilderTestConn{}
	execCtx := legacyBuilderTestContext(t, query, conn)
	images, err := builder.BeforeImage(context.Background(), execCtx)
	if !assert.NoError(t, err) || !assert.Len(t, images, 1) {
		return
	}
	assert.Len(t, images[0].Rows, 2)
	for i, row := range images[0].Rows {
		if assert.Len(t, row.Columns, 1) {
			assert.Equal(t, int64(i+1), row.Columns[0].Value)
			assert.Equal(t, types.IndexTypePrimaryKey, row.Columns[0].KeyType)
		}
	}
	assert.Equal(t, map[string]struct{}{"T_USER:1,2": {}}, execCtx.TxCtx.LockKeys)
}
