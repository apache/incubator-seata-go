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

package parser

import (
	"testing"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
)

func TestBranchUndoLogMethods(t *testing.T) {
	log := &BranchUndoLog{
		Xid:      "test-xid",
		BranchID: 12345,
		Logs: []*SQLUndoLog{
			{
				SQLType:   SQLType_SQLTypeInsert,
				TableName: "test_table",
			},
		},
	}

	// Test getter methods
	assert.Equal(t, "test-xid", log.GetXid())
	assert.Equal(t, uint64(12345), log.GetBranchID())
	assert.Len(t, log.GetLogs(), 1)

	// Test String method
	str := log.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	log.Reset()
	assert.Empty(t, log.GetXid())
	assert.Equal(t, uint64(0), log.GetBranchID())
	assert.Len(t, log.GetLogs(), 0)

	// Test ProtoMessage method
	log.ProtoMessage()

	// Test ProtoReflect method
	msg := log.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = log.Descriptor()
}

func TestSQLUndoLogMethods(t *testing.T) {
	undoLog := &SQLUndoLog{
		SQLType:   SQLType_SQLTypeUpdate,
		TableName: "users",
		BeforeImage: &RecordImage{
			TableName: "users",
			SQLType:   SQLType_SQLTypeUpdate,
		},
		AfterImage: &RecordImage{
			TableName: "users",
			SQLType:   SQLType_SQLTypeUpdate,
		},
	}

	// Test getter methods
	assert.Equal(t, SQLType_SQLTypeUpdate, undoLog.GetSQLType())
	assert.Equal(t, "users", undoLog.GetTableName())
	assert.NotNil(t, undoLog.GetBeforeImage())
	assert.NotNil(t, undoLog.GetAfterImage())

	// Test String method
	str := undoLog.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	undoLog.Reset()
	assert.Equal(t, SQLType_SQLTypeSelect, undoLog.GetSQLType()) // Default enum value
	assert.Empty(t, undoLog.GetTableName())

	// Test ProtoMessage method
	undoLog.ProtoMessage()

	// Test ProtoReflect method
	msg := undoLog.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = undoLog.Descriptor()
}

func TestRecordImageMethods(t *testing.T) {
	recordImage := &RecordImage{
		Index:     1,
		TableName: "test_table",
		SQLType:   SQLType_SQLTypeInsert,
		Rows: []*RowImage{
			{
				Columns: []*ColumnImage{
					{
						KeyType:    IndexType_IndexTypePrimaryKey,
						ColumnName: "id",
						ColumnType: JDBCType_JDBCTypeBigInt,
					},
				},
			},
		},
		TableMeta: &TableMeta{
			TableName: "test_table",
		},
	}

	// Test getter methods
	assert.Equal(t, int32(1), recordImage.GetIndex()) // Use int32 instead of uint32
	assert.Equal(t, "test_table", recordImage.GetTableName())
	assert.Equal(t, SQLType_SQLTypeInsert, recordImage.GetSQLType())
	assert.Len(t, recordImage.GetRows(), 1)
	assert.NotNil(t, recordImage.GetTableMeta())

	// Test String method
	str := recordImage.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	recordImage.Reset()
	assert.Equal(t, int32(0), recordImage.GetIndex()) // Use int32 instead of uint32
	assert.Empty(t, recordImage.GetTableName())

	// Test ProtoMessage method
	recordImage.ProtoMessage()

	// Test ProtoReflect method
	msg := recordImage.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = recordImage.Descriptor()
}

func TestRowImageMethods(t *testing.T) {
	rowImage := &RowImage{
		Columns: []*ColumnImage{
			{
				KeyType:    IndexType_IndexTypePrimaryKey,
				ColumnName: "id",
				ColumnType: JDBCType_JDBCTypeBigInt,
				Value:      &any.Any{},
			},
		},
	}

	// Test getter methods
	assert.Len(t, rowImage.GetColumns(), 1)

	// Test String method
	str := rowImage.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	rowImage.Reset()
	assert.Len(t, rowImage.GetColumns(), 0)

	// Test ProtoMessage method
	rowImage.ProtoMessage()

	// Test ProtoReflect method
	msg := rowImage.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = rowImage.Descriptor()
}

func TestColumnImageMethods(t *testing.T) {
	columnImage := &ColumnImage{
		KeyType:    IndexType_IndexTypePrimaryKey,
		ColumnName: "id",
		ColumnType: JDBCType_JDBCTypeBigInt,
		Value:      &any.Any{},
	}

	// Test getter methods
	assert.Equal(t, IndexType_IndexTypePrimaryKey, columnImage.GetKeyType())
	assert.Equal(t, "id", columnImage.GetColumnName())
	assert.Equal(t, JDBCType_JDBCTypeBigInt, columnImage.GetColumnType())
	assert.NotNil(t, columnImage.GetValue())

	// Test String method
	str := columnImage.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	columnImage.Reset()
	assert.Equal(t, IndexType_IndexTypeNull, columnImage.GetKeyType()) // Default enum value
	assert.Empty(t, columnImage.GetColumnName())

	// Test ProtoMessage method
	columnImage.ProtoMessage()

	// Test ProtoReflect method
	msg := columnImage.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = columnImage.Descriptor()
}

func TestTableMetaMethods(t *testing.T) {
	tableMeta := &TableMeta{
		TableName:   "test_table",
		ColumnNames: []string{"id", "name"},
	}

	// Test getter methods
	assert.Equal(t, "test_table", tableMeta.GetTableName())
	assert.Len(t, tableMeta.GetColumnNames(), 2)
	// These return nil when empty, so test appropriately
	columns := tableMeta.GetColumns()
	indexs := tableMeta.GetIndexs()
	_ = columns // May be nil, just test it doesn't panic
	_ = indexs  // May be nil, just test it doesn't panic

	// Test String method
	str := tableMeta.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	tableMeta.Reset()
	assert.Empty(t, tableMeta.GetTableName())

	// Test ProtoMessage method
	tableMeta.ProtoMessage()

	// Test ProtoReflect method
	msg := tableMeta.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = tableMeta.Descriptor()
}

func TestColumnMetaMethods(t *testing.T) {
	columnMeta := &ColumnMeta{
		Schema:             "test_schema",
		Table:              "test_table",
		ColumnName:         "id",
		ColumnType:         "bigint",
		DatabaseTypeString: "mysql",
	}

	// Test getter methods
	assert.Equal(t, "test_schema", columnMeta.GetSchema())
	assert.Equal(t, "test_table", columnMeta.GetTable())
	assert.Equal(t, "id", columnMeta.GetColumnName())
	assert.Equal(t, "bigint", columnMeta.GetColumnType())
	assert.Equal(t, "mysql", columnMeta.GetDatabaseTypeString())
	// GetColumnDef returns []byte which may be nil when empty
	columnDef := columnMeta.GetColumnDef()
	_ = columnDef // May be nil, just test it doesn't panic
	assert.Equal(t, false, columnMeta.GetAutoincrement())
	assert.Equal(t, int32(0), columnMeta.GetDatabaseType())
	assert.Empty(t, columnMeta.GetColumnKey())
	assert.Equal(t, int32(0), columnMeta.GetIsNullable())
	assert.Empty(t, columnMeta.GetExtra())

	// Test String method
	str := columnMeta.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	columnMeta.Reset()
	assert.Empty(t, columnMeta.GetSchema())
	assert.Empty(t, columnMeta.GetTable())

	// Test ProtoMessage method
	columnMeta.ProtoMessage()

	// Test ProtoReflect method
	msg := columnMeta.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = columnMeta.Descriptor()
}

func TestIndexMetaMethods(t *testing.T) {
	indexMeta := &IndexMeta{
		Schema:     "test_schema",
		Table:      "test_table",
		Name:       "pk_id",
		ColumnName: "id",
		IType:      IndexType_IndexTypePrimaryKey,
		Columns: []*ColumnMeta{
			{
				ColumnName: "id",
			},
		},
	}

	// Test getter methods
	assert.Equal(t, "test_schema", indexMeta.GetSchema())
	assert.Equal(t, "test_table", indexMeta.GetTable())
	assert.Equal(t, "pk_id", indexMeta.GetName())
	assert.Equal(t, "id", indexMeta.GetColumnName())
	assert.Equal(t, IndexType_IndexTypePrimaryKey, indexMeta.GetIType())
	assert.Len(t, indexMeta.GetColumns(), 1)

	// Test String method
	str := indexMeta.String()
	assert.NotEmpty(t, str)

	// Test Reset method
	indexMeta.Reset()
	assert.Empty(t, indexMeta.GetSchema())
	assert.Empty(t, indexMeta.GetTable())

	// Test ProtoMessage method
	indexMeta.ProtoMessage()

	// Test ProtoReflect method
	msg := indexMeta.ProtoReflect()
	assert.NotNil(t, msg)

	// Test Descriptor method
	_, _ = indexMeta.Descriptor()
}

func TestEnumMethods(t *testing.T) {
	// Test SQLType enum methods
	sqlType := SQLType_SQLTypeInsert
	assert.Equal(t, "SQLTypeInsert", sqlType.String())
	assert.Equal(t, int32(1), int32(sqlType.Number()))
	assert.NotNil(t, sqlType.Type())
	assert.NotNil(t, sqlType.Descriptor())

	// Test IndexType enum methods
	indexType := IndexType_IndexTypePrimaryKey
	assert.Equal(t, "IndexTypePrimaryKey", indexType.String())
	assert.Equal(t, int32(1), int32(indexType.Number()))
	assert.NotNil(t, indexType.Type())
	assert.NotNil(t, indexType.Descriptor())

	// Test JDBCType enum methods
	jdbcType := JDBCType_JDBCTypeBigInt
	assert.Equal(t, "JDBCTypeBigInt", jdbcType.String())
	assert.Equal(t, int32(-5), int32(jdbcType.Number()))
	assert.NotNil(t, jdbcType.Type())
	assert.NotNil(t, jdbcType.Descriptor())
}
