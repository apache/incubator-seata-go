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
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/goccy/go-json"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/log"
)

var _ undo.UndoExecutor = (*BaseExecutor)(nil)

const (
	checkSQLTemplate = "SELECT * FROM %s WHERE %s FOR UPDATE"
	maxInSize        = 1000
)

type BaseExecutor struct {
	sqlUndoLog undo.SQLUndoLog
	undoImage  *types.RecordImage
}

// ExecuteOn
func (b *BaseExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
	// check data if valid
	return nil
}

// UndoPrepare
func (b *BaseExecutor) UndoPrepare(undoPST *sql.Stmt, undoValues []types.ColumnImage, pkValueList []types.ColumnImage) {

}

func (b *BaseExecutor) dataValidationAndGoOn(ctx context.Context, conn *sql.Conn) (bool, error) {
	if !undo.UndoConfig.DataValidation {
		return true, nil
	}
	beforeImage := b.sqlUndoLog.BeforeImage
	afterImage := b.sqlUndoLog.AfterImage

	equals, err := IsRecordsEquals(beforeImage, afterImage)
	if err != nil {
		return false, err
	}
	if equals {
		log.Infof("Stop rollback because there is no data change between the before data snapshot and the after data snapshot.")
		return false, nil
	}

	// Validate if data is dirty.
	currentImage, err := b.queryCurrentRecords(ctx, conn)
	if err != nil {
		return false, err
	}
	// compare with current data and after image.
	equals, err = IsRecordsEquals(afterImage, currentImage)
	if err != nil {
		return false, err
	}
	if !equals {
		// If current data is not equivalent to the after data, then compare the current data with the before
		// data, too. No need continue to undo if current data is equivalent to the before data snapshot
		equals, err = IsRecordsEquals(beforeImage, currentImage)
		if err != nil {
			return false, err
		}

		if equals {
			log.Infof("Stop rollback because there is no data change between the before data snapshot and the current data snapshot.")
			// no need continue undo.
			return false, nil
		} else {
			oldRowJson, _ := json.Marshal(afterImage.Rows)
			newRowJson, _ := json.Marshal(currentImage.Rows)
			log.Infof("check dirty data failed, old and new data are not equal, "+
				"tableName:[%s], oldRows:[%s],newRows:[%s].", afterImage.TableName, oldRowJson, newRowJson)
			return false, fmt.Errorf("Has dirty records when undo.")
		}
	}
	return true, nil
}

func (b *BaseExecutor) queryCurrentRecords(ctx context.Context, conn *sql.Conn) (*types.RecordImage, error) {
	if b.undoImage == nil {
		return nil, fmt.Errorf("undo image is nil")
	}
	tableMeta := b.undoImage.TableMeta
	pkNameList := tableMeta.GetPrimaryKeyOnlyName()
	pkValues := b.parsePkValues(b.undoImage.Rows, pkNameList)

	if len(pkValues) == 0 {
		return nil, nil
	}

	where := buildWhereConditionByPKs(pkNameList, len(b.undoImage.Rows), maxInSize)
	checkSQL := fmt.Sprintf(checkSQLTemplate, b.undoImage.TableName, where)
	params := buildPKParams(b.undoImage.Rows, pkNameList)

	rows, err := conn.QueryContext(ctx, checkSQL, params...)
	if err != nil {
		return nil, err
	}

	image := types.RecordImage{
		TableName: b.undoImage.TableName,
		TableMeta: tableMeta,
		SQLType:   types.SQLTypeSelect,
	}
	rowImages := make([]types.RowImage, 0)
	for rows.Next() {
		columnTypes, err := rows.ColumnTypes()
		if err != nil {
			return nil, err
		}
		slice := datasource.GetScanSlice(columnTypes)
		if err = rows.Scan(slice...); err != nil {
			return nil, err
		}

		colNames, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		columns := make([]types.ColumnImage, 0)
		for i, val := range slice {
			actualVal := val
			if v, ok := val.(driver.Valuer); ok {
				actualVal, _ = v.Value()
			}
			columns = append(columns, types.ColumnImage{
				ColumnName: colNames[i],
				Value:      actualVal,
			})
		}
		rowImages = append(rowImages, types.RowImage{Columns: columns})
	}

	image.Rows = rowImages
	return &image, nil
}

func (b *BaseExecutor) parsePkValues(rows []types.RowImage, pkNameList []string) map[string][]types.ColumnImage {
	pkValues := make(map[string][]types.ColumnImage)
	// todo optimize 3 fors
	for _, row := range rows {
		for _, column := range row.Columns {
			for _, pk := range pkNameList {
				if strings.EqualFold(pk, column.ColumnName) {
					values := pkValues[strings.ToUpper(pk)]
					if values == nil {
						values = make([]types.ColumnImage, 0)
					}
					values = append(values, column)
					pkValues[pk] = values
				}
			}
		}
	}
	return pkValues
}
