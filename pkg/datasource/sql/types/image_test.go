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
