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

package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	serr "seata.apache.org/seata-go/pkg/util/errors"
	"seata.apache.org/seata-go/pkg/util/log"
)

type testableBaseExecutor struct {
	BaseExecutor
	mockCurrentImage *types.RecordImage
}

func (t *testableBaseExecutor) queryCurrentRecords(ctx context.Context, conn *sql.Conn) (*types.RecordImage, error) {
	return t.mockCurrentImage, nil
}

func (t *testableBaseExecutor) dataValidationAndGoOn(ctx context.Context, conn *sql.Conn) (bool, error) {
	if !undo.UndoConfig.DataValidation {
		return true, nil
	}
	beforeImage := t.sqlUndoLog.BeforeImage
	afterImage := t.sqlUndoLog.AfterImage

	equals, err := IsRecordsEquals(beforeImage, afterImage)
	if err != nil {
		return false, err
	}
	if equals {
		log.Infof("Stop rollback because there is no data change between the before data snapshot and the after data snapshot.")
		return false, nil
	}

	currentImage, err := t.queryCurrentRecords(ctx, conn)
	if err != nil {
		return false, err
	}

	equals, err = IsRecordsEquals(afterImage, currentImage)
	if err != nil {
		return false, err
	}
	if !equals {
		equals, err = IsRecordsEquals(beforeImage, currentImage)
		if err != nil {
			return false, err
		}
		if equals {
			log.Infof("Stop rollback because there is no data change between the before data snapshot and the current data snapshot.")
			return false, nil
		} else {
			oldRowJson, _ := json.Marshal(afterImage.Rows)
			newRowJson, _ := json.Marshal(currentImage.Rows)
			log.Infof("check dirty data failed, old and new data are not equal, "+
				"tableName:[%s], oldRows:[%s],newRows:[%s].", afterImage.TableName, oldRowJson, newRowJson)
			return false, serr.New(serr.SQLUndoDirtyError, "has dirty records when undo", nil)
		}
	}
	return true, nil
}
func TestDataValidationAndGoOn(t *testing.T) {
	tests := []struct {
		name         string
		beforeImage  *types.RecordImage
		afterImage   *types.RecordImage
		currentImage *types.RecordImage
		want         bool
		wantErr      bool
	}{
		{
			name: "before == after, skip rollback",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "after == current, continue rollback",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			currentImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "current == before, rollback already done",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			currentImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "dirty data",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			currentImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "c"},
					}},
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// patch UndoConfig
			cfgPatch := gomonkey.ApplyGlobalVar(&undo.UndoConfig, undo.Config{DataValidation: true})
			defer cfgPatch.Reset()

			// patch IsRecordsEquals
			comparePatch := gomonkey.ApplyFunc(IsRecordsEquals, func(a, b *types.RecordImage) (bool, error) {
				aj, _ := json.Marshal(a.Rows)
				bj, _ := json.Marshal(b.Rows)
				return string(aj) == string(bj), nil
			})
			defer comparePatch.Reset()

			executor := &testableBaseExecutor{
				BaseExecutor: BaseExecutor{
					sqlUndoLog: undo.SQLUndoLog{
						BeforeImage: tt.beforeImage,
						AfterImage:  tt.afterImage,
					},
					undoImage: tt.afterImage,
				},
				mockCurrentImage: tt.currentImage,
			}

			got, err := executor.dataValidationAndGoOn(context.Background(), nil)

			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				var be *serr.SeataError
				if errors.As(err, &be) {
					assert.Equal(t, serr.SQLUndoDirtyError, be.Code)
				} else {
					t.Errorf("expected BusinessError, got: %v", err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
