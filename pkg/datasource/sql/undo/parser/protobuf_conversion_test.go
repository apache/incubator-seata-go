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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestConvertToProto(t *testing.T) {
	tests := []struct {
		name        string
		intreeLog   *undo.BranchUndoLog
		expectError bool
	}{
		{
			name: "convert simple branch undo log",
			intreeLog: &undo.BranchUndoLog{
				Xid:      "test-xid-123",
				BranchID: 456789,
				Logs: []undo.SQLUndoLog{
					{
						SQLType:   types.SQLTypeInsert,
						TableName: "test_table",
						AfterImage: &types.RecordImage{
							TableName: "test_table",
							SQLType:   types.SQLTypeInsert,
							Rows: []types.RowImage{
								{
									Columns: []types.ColumnImage{
										{
											KeyType:    types.PrimaryKey.Number(),
											ColumnName: "id",
											ColumnType: types.JDBCTypeBigInt,
											Value:      int64(123),
										},
										{
											KeyType:    types.IndexTypeNull,
											ColumnName: "name",
											ColumnType: types.JDBCTypeVarchar,
											Value:      "test_name",
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert with before image",
			intreeLog: &undo.BranchUndoLog{
				Xid:      "test-xid-456",
				BranchID: 789123,
				Logs: []undo.SQLUndoLog{
					{
						SQLType:   types.SQLTypeUpdate,
						TableName: "user_table",
						BeforeImage: &types.RecordImage{
							TableName: "user_table",
							SQLType:   types.SQLTypeUpdate,
							Rows: []types.RowImage{
								{
									Columns: []types.ColumnImage{
										{
											KeyType:    types.PrimaryKey.Number(),
											ColumnName: "user_id",
											ColumnType: types.JDBCTypeBigInt,
											Value:      int64(999),
										},
										{
											KeyType:    types.IndexTypeNull,
											ColumnName: "username",
											ColumnType: types.JDBCTypeVarchar,
											Value:      "old_username",
										},
									},
								},
							},
						},
						AfterImage: &types.RecordImage{
							TableName: "user_table",
							SQLType:   types.SQLTypeUpdate,
							Rows: []types.RowImage{
								{
									Columns: []types.ColumnImage{
										{
											KeyType:    types.PrimaryKey.Number(),
											ColumnName: "user_id",
											ColumnType: types.JDBCTypeBigInt,
											Value:      int64(999),
										},
										{
											KeyType:    types.IndexTypeNull,
											ColumnName: "username",
											ColumnType: types.JDBCTypeVarchar,
											Value:      "new_username",
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert with nil images",
			intreeLog: &undo.BranchUndoLog{
				Xid:      "test-xid-789",
				BranchID: 111222,
				Logs: []undo.SQLUndoLog{
					{
						SQLType:     types.SQLTypeDelete,
						TableName:   "delete_table",
						BeforeImage: nil,
						AfterImage:  nil,
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert with multiple logs",
			intreeLog: &undo.BranchUndoLog{
				Xid:      "test-xid-multi",
				BranchID: 333444,
				Logs: []undo.SQLUndoLog{
					{
						SQLType:   types.SQLTypeInsert,
						TableName: "table1",
					},
					{
						SQLType:   types.SQLTypeUpdate,
						TableName: "table2",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protoLog := ConvertToProto(tt.intreeLog)

			assert.NotNil(t, protoLog)
			assert.Equal(t, tt.intreeLog.Xid, protoLog.Xid)
			assert.Equal(t, tt.intreeLog.BranchID, protoLog.BranchID)
			assert.Equal(t, len(tt.intreeLog.Logs), len(protoLog.Logs))

			for i, sqlLog := range tt.intreeLog.Logs {
				protoSqlLog := protoLog.Logs[i]
				assert.Equal(t, SQLType(sqlLog.SQLType), protoSqlLog.SQLType)
				assert.Equal(t, sqlLog.TableName, protoSqlLog.TableName)

				if sqlLog.BeforeImage != nil {
					assert.NotNil(t, protoSqlLog.BeforeImage)
					assert.Equal(t, sqlLog.BeforeImage.TableName, protoSqlLog.BeforeImage.TableName)
					assert.Equal(t, SQLType(sqlLog.BeforeImage.SQLType), protoSqlLog.BeforeImage.SQLType)
				} else {
					assert.Nil(t, protoSqlLog.BeforeImage)
				}

				if sqlLog.AfterImage != nil {
					assert.NotNil(t, protoSqlLog.AfterImage)
					assert.Equal(t, sqlLog.AfterImage.TableName, protoSqlLog.AfterImage.TableName)
					assert.Equal(t, SQLType(sqlLog.AfterImage.SQLType), protoSqlLog.AfterImage.SQLType)
				} else {
					assert.Nil(t, protoSqlLog.AfterImage)
				}
			}
		})
	}
}

func TestConvertToIntree(t *testing.T) {
	tests := []struct {
		name           string
		protoLog       *BranchUndoLog
		expectXid      string
		expectBranchID uint64
	}{
		{
			name: "convert simple protobuf log",
			protoLog: &BranchUndoLog{
				Xid:      "proto-xid-123",
				BranchID: 654321,
				Logs: []*SQLUndoLog{
					{
						SQLType:   SQLType_SQLTypeInsert,
						TableName: "proto_table",
						AfterImage: &RecordImage{
							TableName: "proto_table",
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
						},
					},
				},
			},
			expectXid:      "proto-xid-123",
			expectBranchID: 654321,
		},
		{
			name: "convert with multiple logs",
			protoLog: &BranchUndoLog{
				Xid:      "proto-xid-multi",
				BranchID: 777888,
				Logs: []*SQLUndoLog{
					{
						SQLType:   SQLType_SQLTypeUpdate,
						TableName: "table_a",
						BeforeImage: &RecordImage{
							TableName: "table_a",
							SQLType:   SQLType_SQLTypeUpdate,
						},
					},
					{
						SQLType:   SQLType_SQLTypeDelete,
						TableName: "table_b",
					},
				},
			},
			expectXid:      "proto-xid-multi",
			expectBranchID: 777888,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intreeLog := ConvertToIntree(tt.protoLog)

			assert.NotNil(t, intreeLog)
			assert.Equal(t, tt.expectXid, intreeLog.Xid)
			assert.Equal(t, tt.expectBranchID, intreeLog.BranchID)
			assert.Equal(t, len(tt.protoLog.Logs), len(intreeLog.Logs))

			for i, protoSqlLog := range tt.protoLog.Logs {
				intreeSqlLog := intreeLog.Logs[i]
				assert.Equal(t, types.SQLType(protoSqlLog.SQLType), intreeSqlLog.SQLType)
				assert.Equal(t, protoSqlLog.TableName, intreeSqlLog.TableName)

				if protoSqlLog.BeforeImage != nil {
					assert.NotNil(t, intreeSqlLog.BeforeImage)
					assert.Equal(t, protoSqlLog.BeforeImage.TableName, intreeSqlLog.BeforeImage.TableName)
					assert.Equal(t, types.SQLType(protoSqlLog.BeforeImage.SQLType), intreeSqlLog.BeforeImage.SQLType)
				} else {
					assert.Nil(t, intreeSqlLog.BeforeImage)
				}

				if protoSqlLog.AfterImage != nil {
					assert.NotNil(t, intreeSqlLog.AfterImage)
					assert.Equal(t, protoSqlLog.AfterImage.TableName, intreeSqlLog.AfterImage.TableName)
					assert.Equal(t, types.SQLType(protoSqlLog.AfterImage.SQLType), intreeSqlLog.AfterImage.SQLType)
				} else {
					assert.Nil(t, intreeSqlLog.AfterImage)
				}
			}
		})
	}
}

func TestConvertToProtoAndBack(t *testing.T) {
	originalLog := &undo.BranchUndoLog{
		Xid:      "round-trip-test",
		BranchID: 999888,
		Logs: []undo.SQLUndoLog{
			{
				SQLType:   types.SQLTypeUpdate,
				TableName: "round_trip_table",
				BeforeImage: &types.RecordImage{
					TableName: "round_trip_table",
					SQLType:   types.SQLTypeUpdate,
					Rows: []types.RowImage{
						{
							Columns: []types.ColumnImage{
								{
									KeyType:    types.PrimaryKey.Number(),
									ColumnName: "id",
									ColumnType: types.JDBCTypeBigInt,
									Value:      int64(111),
								},
								{
									KeyType:    types.IndexTypeNull,
									ColumnName: "status",
									ColumnType: types.JDBCTypeVarchar,
									Value:      "old_status",
								},
							},
						},
					},
				},
				AfterImage: &types.RecordImage{
					TableName: "round_trip_table",
					SQLType:   types.SQLTypeUpdate,
					Rows: []types.RowImage{
						{
							Columns: []types.ColumnImage{
								{
									KeyType:    types.PrimaryKey.Number(),
									ColumnName: "id",
									ColumnType: types.JDBCTypeBigInt,
									Value:      int64(111),
								},
								{
									KeyType:    types.IndexTypeNull,
									ColumnName: "status",
									ColumnType: types.JDBCTypeVarchar,
									Value:      "new_status",
								},
							},
						},
					},
				},
			},
		},
	}

	protoLog := ConvertToProto(originalLog)
	convertedBack := ConvertToIntree(protoLog)

	assert.Equal(t, originalLog.Xid, convertedBack.Xid)
	assert.Equal(t, originalLog.BranchID, convertedBack.BranchID)
	assert.Equal(t, len(originalLog.Logs), len(convertedBack.Logs))
}

func TestConvertAnyToInterface(t *testing.T) {
	testValues := []interface{}{
		"string_value",
		int64(12345),
		float64(123.45),
		true,
		nil,
	}

	for i, value := range testValues {
		t.Run("test_convert_any", func(t *testing.T) {
			anyValue, err := convertInterfaceToAny(value)
			require.NoError(t, err)
			require.NotNil(t, anyValue)

			convertedBack, err := convertAnyToInterface(anyValue)
			if value == nil {
				assert.NoError(t, err)
				assert.Nil(t, convertedBack)
			} else {
				assert.NoError(t, err)
				// JSON serialization converts int64 to float64, so we need to handle this
				if i == 1 { // int64(12345) case
					// Check if the value is the expected float64 representation
					if floatVal, ok := convertedBack.(float64); ok {
						assert.Equal(t, float64(12345), floatVal)
					} else {
						// If it's still int64, that's also acceptable
						assert.Equal(t, value, convertedBack)
					}
				} else {
					assert.Equal(t, value, convertedBack)
				}
			}
		})
	}
}

func TestConvertInterfaceToAnyWithComplexTypes(t *testing.T) {
	complexValues := []interface{}{
		map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
		[]interface{}{1, 2, 3, "test"},
		struct {
			Name string
			Age  int
		}{Name: "test", Age: 25},
	}

	for i, value := range complexValues {
		t.Run("test_complex_convert", func(t *testing.T) {
			anyValue, err := convertInterfaceToAny(value)
			assert.NoError(t, err, "Failed to convert complex value %d", i)
			assert.NotNil(t, anyValue)

			convertedBack, err := convertAnyToInterface(anyValue)
			assert.NoError(t, err)
			assert.NotNil(t, convertedBack)
		})
	}
}
