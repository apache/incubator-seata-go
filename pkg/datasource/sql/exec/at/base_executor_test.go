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

package at

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource/mysql"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

type tableMetaReaderForTest struct {
	key      types.TableMetaKey
	meta     *types.TableMeta
	resolved types.TableRef
	readKey  types.TableMetaKey
}

func (r *tableMetaReaderForTest) ResolveTableMetaKey(_ context.Context, _ driver.Conn, ref types.TableRef) (types.TableMetaKey, error) {
	r.resolved = ref
	return r.key, nil
}

func (r *tableMetaReaderForTest) GetTableMeta(_ context.Context, key types.TableMetaKey) (*types.TableMeta, error) {
	r.readKey = key
	return r.meta, nil
}

func TestBaseExecutorUsesExecutionReaderAndResolvedKey(t *testing.T) {
	parsed, err := parser.DoParser("UPDATE shop.orders AS o SET o.id=1")
	assert.NoError(t, err)
	reader := &tableMetaReaderForTest{key: types.TableMetaKey{DBName: "shop", TableName: "orders"}, meta: &types.TableMeta{TableName: "orders"}}
	execCtx := &types.ExecContext{ParseContext: parsed, TableMetaReader: reader}
	exec := &baseExecutor{}
	assert.NoError(t, exec.resolveTableMetaKey(context.Background(), execCtx, parsed))
	meta, err := exec.getTableMeta(context.Background(), execCtx, parsed)
	assert.NoError(t, err)
	assert.Same(t, reader.meta, meta)
	assert.Equal(t, types.TableRef{Qualifier: "shop", TableName: "orders"}, reader.resolved)
	assert.Equal(t, reader.key, reader.readKey)
	assert.Equal(t, &reader.key, execCtx.TableMetaKey)
}

func TestBaseExecutorMissingReaderFailsClearly(t *testing.T) {
	parsed, err := parser.DoParser("UPDATE orders SET id=1")
	assert.NoError(t, err)
	execCtx := &types.ExecContext{ParseContext: parsed}
	exec := &baseExecutor{}
	err = exec.resolveTableMetaKey(context.Background(), execCtx, parsed)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "table meta reader"))
}

func TestQualifiedTableNameUsesResolvedIdentity(t *testing.T) {
	assert.Equal(t, "`shop`.`orders`", qualifiedTableName(&types.TableMetaKey{DBName: "shop", TableName: "orders"}, "orders", types.DBTypeMySQL))
	assert.Equal(t, `"Space"."Users"`, qualifiedTableName(&types.TableMetaKey{Schema: "Space", TableName: "Users"}, "Users", types.DBTypePostgreSQL))
	assert.Equal(t, "orders", qualifiedTableName(nil, "orders", types.DBTypeMySQL))
}

func TestGetScanSlicePreservesDecimal(t *testing.T) {
	executor := baseExecutor{}
	meta := &types.TableMeta{Columns: map[string]types.ColumnMeta{
		"amount": {ColumnName: "amount", DatabaseTypeString: "DECIMAL"},
	}}

	scanSlice := executor.GetScanSlice([]string{"amount"}, meta)

	assert.IsType(t, &sql.NullString{}, scanSlice[0])
}

func TestBaseExecBuildLockKey(t *testing.T) {
	var exec baseExecutor

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
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "user1")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "user2")}},
				},
			},
			"TEST_NAME:1_user1,2_user2",
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
					{Columns: []types.ColumnImage{getColumnImage("name", "mike"), getColumnImage("age", 25)}},
				},
			},
			expected: "MIXED_KEY:mike_25",
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
					{Columns: []types.ColumnImage{getColumnImage("id", "A,b_c")}},
				},
			},
			expected: "SPECIAL:A,b_c",
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
		{
			name: "Multiple Rows With Nil PK Value",
			metaData: types.TableMeta{
				TableName: "nil_pk",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "nil_pk",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", nil)}},
					{Columns: []types.ColumnImage{getColumnImage("id", 123)}},
					{Columns: []types.ColumnImage{getColumnImage("id", nil)}},
				},
			},
			expected: "NIL_PK:,123,",
		},
		{
			name: "PK As Bool And Float",
			metaData: types.TableMeta{
				TableName: "type_pk",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnName, columnAge}},
				},
			},
			records: types.RecordImage{
				TableName: "type_pk",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("name", true), getColumnImage("age", 3.14)}},
					{Columns: []types.ColumnImage{getColumnImage("name", false), getColumnImage("age", 0.0)}},
				},
			},
			expected: "TYPE_PK:true_3.14,false_0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := exec.buildLockKey(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}

func TestBaseExecutorPrepareUndoPair(t *testing.T) {
	meta := &types.TableMeta{
		TableName: "test_table",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType:   types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
			},
		},
	}
	rows := func(id int) []types.RowImage {
		return []types.RowImage{{Columns: []types.ColumnImage{{
			ColumnName: "id",
			KeyType:    types.IndexTypePrimaryKey,
			Value:      id,
		}}}}
	}

	tests := []struct {
		name       string
		sqlType    types.SQLType
		beforeRows []types.RowImage
		afterRows  []types.RowImage
		wantLock   string
		wantPairs  int
	}{
		{name: "empty update", sqlType: types.SQLTypeUpdate, wantPairs: 0},
		{name: "unchanged update", sqlType: types.SQLTypeUpdate, beforeRows: rows(1), afterRows: rows(1), wantPairs: 0},
		{name: "reordered unchanged update", sqlType: types.SQLTypeUpdate, beforeRows: append(rows(1), rows(2)...), afterRows: append(rows(2), rows(1)...), wantPairs: 0},
		{name: "update locks after image", sqlType: types.SQLTypeUpdate, beforeRows: rows(1), afterRows: rows(2), wantLock: "TEST_TABLE:2", wantPairs: 1},
		{name: "delete locks before image", sqlType: types.SQLTypeDelete, beforeRows: rows(1), wantLock: "TEST_TABLE:1", wantPairs: 1},
		{name: "insert locks after image", sqlType: types.SQLTypeInsert, afterRows: rows(2), wantLock: "TEST_TABLE:2", wantPairs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txCtx := types.NewTxCtx()
			execCtx := &types.ExecContext{TxCtx: txCtx}
			beforeImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: tt.sqlType, Rows: tt.beforeRows}
			afterImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: tt.sqlType, Rows: tt.afterRows}

			err := (&baseExecutor{}).prepareUndoPair(execCtx, beforeImage, afterImage)

			assert.NoError(t, err)
			assert.Len(t, txCtx.RoundImages.BeofreImages(), tt.wantPairs)
			assert.Len(t, txCtx.RoundImages.AfterImages(), tt.wantPairs)
			assert.Len(t, txCtx.LockKeys, tt.wantPairs)
			if tt.wantLock != "" {
				assert.Contains(t, txCtx.LockKeys, tt.wantLock)
			}
		})
	}
}

func TestBaseExecutorPrepareUndoPairKeepsCompositeLockKeyFormat(t *testing.T) {
	meta := &types.TableMeta{
		TableName:   "test_table",
		ColumnNames: []string{"id", "tenant_id", "value"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}, {ColumnName: "tenant_id"}},
		}},
	}
	beforeRows := []types.RowImage{{Columns: []types.ColumnImage{
		{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
		{ColumnName: "tenant_id", KeyType: types.IndexTypePrimaryKey, Value: 2},
		{ColumnName: "value", Value: "before"},
	}}}
	afterRows := []types.RowImage{{Columns: []types.ColumnImage{
		{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
		{ColumnName: "tenant_id", KeyType: types.IndexTypePrimaryKey, Value: 2},
		{ColumnName: "value", Value: "after"},
	}}}
	txCtx := types.NewTxCtx()
	execCtx := &types.ExecContext{TxCtx: txCtx, DBType: types.DBTypeMySQL}
	beforeImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: types.SQLTypeUpdate, Rows: beforeRows}
	afterImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: types.SQLTypeUpdate, Rows: afterRows}

	err := (&baseExecutor{}).prepareUndoPair(execCtx, beforeImage, afterImage)

	assert.NoError(t, err)
	assert.Contains(t, txCtx.LockKeys, "TEST_TABLE:1_2")
}

func TestBaseExecutorPrepareUndoPairRejectsInvalidLockImage(t *testing.T) {
	meta := &types.TableMeta{
		TableName: "test_table",
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	primaryKey := func(name string, value interface{}) types.ColumnImage {
		return types.ColumnImage{ColumnName: name, KeyType: types.IndexTypePrimaryKey, Value: value}
	}
	validRows := []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", 1)}}}
	tests := []struct {
		name      string
		meta      *types.TableMeta
		afterRows []types.RowImage
		wantErr   string
	}{
		{name: "chosen lock image is empty", meta: meta, wantErr: "lock image rows are empty"},
		{name: "primary key metadata is empty", meta: &types.TableMeta{TableName: "test_table"}, afterRows: validRows, wantErr: "primary key metadata is empty"},
		{name: "primary key is missing", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{{ColumnName: "name", Value: "test"}}}}, wantErr: "not found in row image"},
		{name: "primary key is duplicated", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", 1), primaryKey("ID", 1)}}}, wantErr: "found more than once"},
		{name: "extra primary key is present", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", 1), primaryKey("tenant_id", 2)}}}, wantErr: "not defined in table metadata"},
		{name: "primary key value is nil", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", nil)}}}, wantErr: "is nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txCtx := types.NewTxCtx()
			execCtx := &types.ExecContext{TxCtx: txCtx, DBType: types.DBTypeMySQL}
			beforeImage := &types.RecordImage{TableName: "test_table", TableMeta: tt.meta, SQLType: types.SQLTypeUpdate, Rows: validRows}
			afterImage := &types.RecordImage{TableName: "test_table", TableMeta: tt.meta, SQLType: types.SQLTypeUpdate, Rows: tt.afterRows}

			err := (&baseExecutor{}).prepareUndoPair(execCtx, beforeImage, afterImage)

			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, txCtx.LockKeys)
			assert.Empty(t, txCtx.RoundImages.BeofreImages())
			assert.Empty(t, txCtx.RoundImages.AfterImages())
		})
	}
}

func TestBuildImageSelectColumns(t *testing.T) {
	compositeMeta := func() *types.TableMeta {
		return &types.TableMeta{
			ColumnNames: []string{"id", "tenant_id", "name"},
			Indexs: map[string]types.IndexMeta{"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", Autoincrement: true},
					{ColumnName: "tenant_id"},
				},
			}},
		}
	}

	tests := []struct {
		name              string
		meta              *types.TableMeta
		requested         []string
		dbType            types.DBType
		onlyCareRequested bool
		want              []string
		wantErr           string
	}{
		{
			name:              "adds only missing composite primary key",
			meta:              compositeMeta(),
			requested:         []string{"tenant_id", "name"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			want:              []string{"tenant_id", "name", "id"},
		},
		{
			name:              "preserves complete composite primary key",
			meta:              compositeMeta(),
			requested:         []string{"tenant_id", "name", "id"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			want:              []string{"tenant_id", "name", "id"},
		},
		{
			name:              "matches escaped primary key case insensitively",
			meta:              compositeMeta(),
			requested:         []string{"`TENANT_ID`", "order"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			want:              []string{"`TENANT_ID`", "`order`", "id"},
		},
		{
			name:              "uses all columns without explicit columns",
			meta:              compositeMeta(),
			dbType:            types.DBTypePostgreSQL,
			onlyCareRequested: true,
			want:              []string{`"id"`, `"tenant_id"`, `"name"`},
		},
		{
			name:              "uses all columns when only care is disabled",
			meta:              compositeMeta(),
			requested:         []string{"name"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: false,
			want:              []string{"id", "tenant_id", "name"},
		},
		{
			name:    "rejects nil metadata",
			dbType:  types.DBTypeMySQL,
			wantErr: "table meta is nil",
		},
		{
			name: "rejects missing primary key metadata",
			meta: &types.TableMeta{
				ColumnNames: []string{"name"},
			},
			dbType:  types.DBTypeMySQL,
			wantErr: "primary key metadata is empty",
		},
		{
			name:              "rejects duplicate requested columns",
			meta:              compositeMeta(),
			requested:         []string{"id", "`ID`"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			wantErr:           "found more than once",
		},
		{
			name: "rejects duplicate primary key metadata",
			meta: &types.TableMeta{
				ColumnNames: []string{"id", "name"},
				Indexs: map[string]types.IndexMeta{"PRIMARY": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
						{ColumnName: "`ID`"},
					},
				}},
			},
			requested:         []string{"name"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			wantErr:           "exists more than once in metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildImageSelectColumns(tt.meta, tt.requested, tt.dbType, tt.onlyCareRequested)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildImageSelectColumnsDoesNotMutateInput(t *testing.T) {
	meta := &types.TableMeta{
		ColumnNames: []string{"id", "name", "age"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	requested := make([]string, 2, 3)
	copy(requested, []string{"name", "age"})

	_, err := buildImageSelectColumns(meta, requested, types.DBTypePostgreSQL, true)

	assert.NoError(t, err)
	assert.Equal(t, []string{"name", "age"}, requested)
}

func TestBaseExecBuildLockKey_EscapedColumnNames(t *testing.T) {
	var exec baseExecutor

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
			name: "Backtick-escaped single PK",
			metaData: types.TableMeta{
				TableName: "test_table",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
				},
			},
			records: types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("`id`", 1), {ColumnName: "`name`", Value: "test"}}},
				},
			},
			expected: "TEST_TABLE:1",
		},
		{
			name: "Backtick-escaped composite PK",
			metaData: types.TableMeta{
				TableName: "orders",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{
						{ColumnName: "order_id"},
						{ColumnName: "user_id"},
					}},
				},
			},
			records: types.RecordImage{
				TableName: "orders",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("`order_id`", 100), getColumnImage("`user_id`", 1)}},
					{Columns: []types.ColumnImage{getColumnImage("`order_id`", 200), getColumnImage("`user_id`", 2)}},
				},
			},
			expected: "ORDERS:100_1,200_2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := exec.buildLockKey(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}

func TestBaseExecutorWriteDatabaseBoundary(t *testing.T) {
	for _, tc := range []struct {
		name       string
		query      string
		currentDB  string
		wantReject bool
	}{
		{"cross database insert", "INSERT INTO db_b.account (id) VALUES (1)", "", true},
		{"cross database update", "UPDATE db_b.account SET balance=1 WHERE id=1", "", true},
		{"cross database delete", "DELETE FROM db_b.account WHERE id=1", "", true},
		{"cross database upsert", "INSERT INTO db_b.account (id) VALUES (1) ON DUPLICATE KEY UPDATE balance=1", "", true},
		{"quoted database", "UPDATE `db_b`.`account` SET balance=1", "", true},
		{"parenthesized table", "UPDATE (db_b.account) SET balance=1", "", true},
		{"case insensitive write alias", "UPDATE db_b.account A SET a.balance=1", "", true},
		{"default database changed", "UPDATE account SET balance=1", "db_b", true},
		{"same database insert", "INSERT INTO db_a.account (id) VALUES (1)", "", false},
		{"same database update", "UPDATE `db_a`.`account` SET balance=1", "", false},
		{"same database delete", "DELETE FROM db_a.account WHERE id=1", "", false},
		{"same database upsert", "INSERT INTO db_a.account (id) VALUES (1) ON DUPLICATE KEY UPDATE balance=1", "", false},
		{"same default database", "UPDATE account SET balance=1", "db_a", false},
		{"insert reads another database", "INSERT INTO db_a.account (id) SELECT id FROM db_b.account", "", false},
		{"predicate reads another database", "UPDATE db_a.account SET balance=1 WHERE id IN (SELECT id FROM db_b.account)", "", false},
		{"ordinary read", "SELECT * FROM db_b.account", "", false},
		{"locking read", "SELECT * FROM db_b.account FOR UPDATE", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkWriteDatabaseBoundary(t, tc.query, tc.currentDB, tc.wantReject)
		})
	}
}

func TestBaseExecutorWriteDatabaseCaseSensitivity(t *testing.T) {
	for _, mode := range []int64{0, 1, 2} {
		for _, query := range []string{
			"INSERT INTO DB_A.account (id) VALUES (1)",
			"UPDATE DB_A.account SET balance=1",
			"DELETE FROM DB_A.account WHERE id=1",
			"INSERT INTO DB_A.account (id) VALUES (1) ON DUPLICATE KEY UPDATE balance=1",
		} {
			t.Run(fmt.Sprintf("mode_%d/%s", mode, query), func(t *testing.T) {
				checkWriteDatabaseBoundary(t, query, "", mode == 0, mode)
			})
		}
		t.Run(fmt.Sprintf("mode_%d/default_database", mode), func(t *testing.T) {
			checkWriteDatabaseBoundary(t, "UPDATE account SET balance=1", "DB_A", mode == 0, mode)
		})
	}
}

// Exercise the real dispatcher and executors. Allowed writes stop at the first
// image/metadata read; rejected writes must fail before reaching that read.
func checkWriteDatabaseBoundary(t *testing.T, query, currentDB string, wantReject bool, caseMode ...int64) {
	t.Helper()
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	if currentDB != "" {
		sqlMock.ExpectQuery("SELECT DATABASE").WillReturnRows(sqlmock.NewRows([]string{"DATABASE()"}).AddRow(currentDB))
	}
	if len(caseMode) != 0 {
		sqlMock.ExpectQuery("SELECT @@lower_case_table_names").
			WillReturnRows(sqlmock.NewRows([]string{"@@lower_case_table_names"}).AddRow(caseMode[0])).RowsWillBeClosed()
	}
	stop := errors.New("database boundary passed, image read reached")
	reader := mock.NewMockTableMetaCache(gomock.NewController(t))
	resolver := mysql.NewTableMetaInstance(nil, nil)
	reader.EXPECT().ResolveTableMetaKey(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(resolver.ResolveTableMetaKey).AnyTimes()
	if !wantReject {
		reader.EXPECT().GetTableMeta(gomock.Any(), gomock.Any()).Return(nil, stop).AnyTimes()
		if strings.HasPrefix(query, "DELETE") {
			sqlMock.ExpectQuery("FOR UPDATE").WillReturnError(stop)
		}
	}
	calls := 0
	err = conn.Raw(func(raw any) error {
		_, err := (&ATExecutor{}).ExecWithNamedValue(context.Background(), &types.ExecContext{
			Query: query, DBType: types.DBTypeMySQL, DBName: "db_a", DbVersion: "8.0.29",
			Conn: raw.(driver.Conn), TableMetaReader: reader, IsRequireGlobalLock: true,
			TxCtx: &types.TransactionContext{TransactionMode: types.ATMode},
		}, func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
			calls++
			return &mockExecResult{rowsAffected: 1}, nil
		})
		return err
	})
	if wantReject {
		require.ErrorContains(t, err, "AT write target database")
		require.ErrorContains(t, err, `resource database "db_a"`)
	} else if query == "SELECT * FROM db_b.account" {
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		return
	} else {
		require.ErrorIs(t, err, stop)
	}
	require.Zero(t, calls)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestWriteDatabaseCasePolicyReadFailure(t *testing.T) {
	for _, failure := range []string{"query", "no rows", "scan", "row"} {
		t.Run(failure, func(t *testing.T) {
			db, sqlMock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			defer conn.Close()
			expected := sqlMock.ExpectQuery("SELECT @@lower_case_table_names")
			switch failure {
			case "query":
				expected.WillReturnError(errors.New("policy unavailable"))
			case "no rows":
				expected.WillReturnRows(sqlmock.NewRows([]string{"mode"})).RowsWillBeClosed()
			case "scan":
				expected.WillReturnRows(sqlmock.NewRows([]string{"mode"}).AddRow(nil)).RowsWillBeClosed()
			case "row":
				expected.WillReturnRows(sqlmock.NewRows([]string{"mode"}).AddRow(1).
					RowError(0, errors.New("policy row unavailable"))).RowsWillBeClosed()
			}
			err = conn.Raw(func(raw any) error {
				return validateWriteDatabase(context.Background(), &types.ExecContext{
					DBType: types.DBTypeMySQL, DBName: "db_a", Conn: raw.(driver.Conn),
					TxCtx: &types.TransactionContext{TransactionMode: types.ATMode},
				}, types.TableMetaKey{DBName: "DB_A", TableName: "account"})
			})
			require.ErrorContains(t, err, "read lower_case_table_names")
			require.NoError(t, sqlMock.ExpectationsWereMet())
		})
	}
}
