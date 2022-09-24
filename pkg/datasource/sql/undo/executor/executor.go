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
	"strings"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/impl"
)

var _ undo.UndoExecutor = (*BaseExecutor)(nil)

type BaseExecutor struct {
}

// ExecuteOn
func (b *BaseExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, sqlUndoLog impl.SQLUndoLog, conn *sql.Conn) error {
	// check data if valid
	// Todo implement dataValidationAndGoOn
	if !b.dataValidationAndGoOn(sqlUndoLog, conn) {
		return nil
	}

	return nil
}

// dataValidationAndGoOn
func (b *BaseExecutor) dataValidationAndGoOn(sqlUndoLog impl.SQLUndoLog, conn *sql.Conn) bool {
	return true
}

// isRecordsEquals
func (b *BaseExecutor) isRecordsEquals(before types.RecordImages, after types.RecordImages) bool {
	lenBefore, lenAfter := len(before), len(after)
	if lenBefore == 0 && lenAfter == 0 {
		return true
	}

	if lenBefore > 0 && lenAfter == 0 || lenBefore == 0 && lenAfter > 0 {
		return false
	}

	for key, _ := range before {
		if strings.EqualFold(before[key].Table, after[key].Table) &&
			len(before[key].Rows) == len(after[key].Rows) {
			// when image is EmptyTableRecords, getTableMeta will throw an exception
			if len(before[key].Rows) == 0 {
				return true
			}

		}
	}

	return true
}

// UndoPrepare
func (b *BaseExecutor) UndoPrepare(undoPST *sql.Stmt, undoValues []types.ColumnImage, pkValueList []types.ColumnImage) {

}

// GetOrderedPkList
func (b *BaseExecutor) GetOrderedPkList(image *types.RecordImage, row types.RowImage, dbType types.DBType) ([]types.ColumnImage, error) {

	pkColumnNameListByOrder, err := image.TableMeta.GetPrimaryKeyOnlyName()
	if err != nil {
		log.Errorf("[GetOrderedPkList] get primary fail, err: %v", err)
		return nil, err
	}

	pkColumnNameListNoOrder := make([]types.ColumnImage, 0)
	pkFields := make([]types.ColumnImage, 0)

	for _, column := range row.PrimaryKeys(row.Columns) {
		column.Name = DelEscape(column.Name, dbType)
		pkColumnNameListNoOrder = append(pkColumnNameListNoOrder, column)
	}

	for _, pkName := range pkColumnNameListByOrder {
		for _, col := range pkColumnNameListNoOrder {
			if strings.Index(col.Name, pkName) > -1 {
				pkFields = append(pkFields, col)
			}
		}
	}

	return pkFields, nil
}
