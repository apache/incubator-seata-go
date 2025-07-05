package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	serr "seata.apache.org/seata-go/pkg/util/errors"
	"seata.apache.org/seata-go/pkg/util/log"
	"testing"
)

type testableBaseExecutor struct {
	BaseExecutor
	mockCurrentImage *types.RecordImage
}

// 重写 queryCurrentRecords，返回 mock 数据
func (t *testableBaseExecutor) queryCurrentRecords(ctx context.Context, conn *sql.Conn) (*types.RecordImage, error) {
	return t.mockCurrentImage, nil
}

// 重写 dataValidationAndGoOn，使其调用子类方法
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

	currentImage, err := t.queryCurrentRecords(ctx, conn) // 👈 会走子类方法
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
						SQLType:     types.SQLTypeUpdate, // 添加这行
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
