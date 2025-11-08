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

package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestColumnImage_UnmarshalJSON(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		image       *ColumnImage
		expectValue interface{}
	}{
		{
			name: "test-string",
			image: &ColumnImage{
				KeyType:    IndexTypePrimaryKey,
				ColumnName: "Name",
				ColumnType: JDBCTypeVarchar,
				Value:      []byte("Seata-go"),
			},
			expectValue: "Seata-go",
		},
		{
			name: "test-text",
			image: &ColumnImage{
				KeyType:    IndexTypePrimaryKey,
				ColumnName: "Text",
				ColumnType: JDBCTypeLongVarchar,
				Value:      []byte("Seata-go"),
			},
			expectValue: "Seata-go",
		},
		{
			name: "test-int",
			image: &ColumnImage{
				KeyType:    IndexTypeNull,
				ColumnName: "Age",
				ColumnType: JDBCTypeTinyInt,
				Value:      int8(20),
			},
			expectValue: int8(20),
		},
		{
			name: "test-double",
			image: &ColumnImage{
				KeyType:    IndexTypeNull,
				ColumnName: "Salary",
				ColumnType: JDBCTypeDouble,
				Value:      8899.778,
			},
			expectValue: float64(8899.778),
		},
		{
			name: "test-time",
			image: &ColumnImage{
				KeyType:    IndexTypeNull,
				ColumnName: "CreateTime",
				ColumnType: JDBCTypeTime,
				Value:      now,
			},
			expectValue: now,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.image)
			assert.Nil(t, err)
			var after ColumnImage
			err = json.Unmarshal(data, &after)
			assert.Nil(t, err)
			if ti, ok := tt.expectValue.(time.Time); ok {
				assert.Equal(t, ti.Format(time.RFC3339Nano), after.Value.(time.Time).Format(time.RFC3339Nano))
			} else {
				assert.Equal(t, tt.expectValue, after.Value)
			}
		})
	}
}

func TestRoundRecordImage_Methods(t *testing.T) {
	before := []*RecordImage{{Rows: []RowImage{{}}}}
	after := []*RecordImage{{Rows: []RowImage{{}}}}

	round := &RoundRecordImage{
		before: before,
		after:  after,
	}

	t.Run("AppendBeofreImages", func(t *testing.T) {
		newImages := []*RecordImage{{Rows: []RowImage{{}}}}
		round.AppendBeofreImages(newImages)
		assert.Len(t, round.before, 2)
	})

	t.Run("AppendBeofreImage", func(t *testing.T) {
		newImage := &RecordImage{Rows: []RowImage{{}}}
		round.AppendBeofreImage(newImage)
		assert.Len(t, round.before, 3)
	})

	t.Run("AppendAfterImages", func(t *testing.T) {
		newImages := []*RecordImage{{Rows: []RowImage{{}}}}
		round.AppendAfterImages(newImages)
		assert.Len(t, round.after, 2)
	})

	t.Run("AppendAfterImage", func(t *testing.T) {
		newImage := &RecordImage{Rows: []RowImage{{}}}
		round.AppendAfterImage(newImage)
		assert.Len(t, round.after, 3)
	})

	t.Run("BeofreImages", func(t *testing.T) {
		result := round.BeofreImages()
		assert.Equal(t, round.before, result)
	})

	t.Run("AfterImages", func(t *testing.T) {
		result := round.AfterImages()
		assert.Equal(t, round.after, result)
	})

	t.Run("IsBeforeAfterSizeEq", func(t *testing.T) {
		result := round.IsBeforeAfterSizeEq()
		assert.True(t, result) // both have 3 elements now
	})
}

func TestRecordImages_Reserve(t *testing.T) {
	images := RecordImages{
		&RecordImage{TableName: "table1"},
		&RecordImage{TableName: "table2"},
		&RecordImage{TableName: "table3"},
	}

	images.Reserve()

	assert.Equal(t, "table3", images[0].TableName)
	assert.Equal(t, "table2", images[1].TableName)
	assert.Equal(t, "table1", images[2].TableName)
}

func TestRecordImages_IsEmptyImage(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		images := RecordImages{}
		assert.True(t, images.IsEmptyImage())
	})

	t.Run("nil slice", func(t *testing.T) {
		var images RecordImages
		assert.True(t, images.IsEmptyImage())
	})

	t.Run("non-empty slice with empty images", func(t *testing.T) {
		images := RecordImages{&RecordImage{}}
		assert.True(t, images.IsEmptyImage())
	})

	t.Run("non-empty slice with non-empty images", func(t *testing.T) {
		images := RecordImages{&RecordImage{Rows: []RowImage{{}}}}
		assert.False(t, images.IsEmptyImage())
	})
}

func TestNewEmptyRecordImage(t *testing.T) {
	tableMeta := &TableMeta{TableName: "test_table"}
	sqlType := SQLType(SQLTypeInsert)
	image := NewEmptyRecordImage(tableMeta, sqlType)

	assert.Equal(t, "test_table", image.TableName)
	assert.Equal(t, sqlType, image.SQLType)
	assert.Equal(t, tableMeta, image.TableMeta)
	assert.Empty(t, image.Rows)
}

func TestRowImage_GetColumnMap(t *testing.T) {
	row := &RowImage{
		Columns: []ColumnImage{
			{ColumnName: "id", Value: 1},
			{ColumnName: "name", Value: "test"},
		},
	}

	columnMap := row.GetColumnMap()
	assert.Len(t, columnMap, 2)
	assert.Equal(t, 1, columnMap["id"].Value)
	assert.Equal(t, "test", columnMap["name"].Value)
}

func TestColumnImage_GetActualValue(t *testing.T) {
	tests := []struct {
		name     string
		column   *ColumnImage
		expected interface{}
	}{
		{
			name: "string value from bytes",
			column: &ColumnImage{
				ColumnType: JDBCTypeVarchar,
				Value:      []byte("test"),
			},
			expected: []byte("test"),
		},
		{
			name: "direct value",
			column: &ColumnImage{
				ColumnType: JDBCTypeInteger,
				Value:      123,
			},
			expected: 123,
		},
		{
			name: "nil value",
			column: &ColumnImage{
				ColumnType: JDBCTypeVarchar,
				Value:      nil,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.column.GetActualValue()
			assert.Equal(t, tt.expected, result)
		})
	}
}
