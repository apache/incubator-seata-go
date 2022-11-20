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

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/util/log"
)

var _ undo.UndoExecutor = (*BaseExecutor)(nil)

const (
	selectSQL = "SELECT * FROM %s WHERE %s FOR UPDATE"
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

func (b *BaseExecutor) dataValidationAndGoOn(conn *sql.Conn) (bool, error) {
	beforeImage := b.sqlUndoLog.BeforeImage
	afterImage := b.sqlUndoLog.AfterImage

	equal, err := IsRecordsEquals(beforeImage, afterImage)
	if err != nil {
		return false, err
	}
	if equal {
		log.Infof("Stop rollback because there is no data change between the before data snapshot and the after data snapshot.")
		return false, nil
	}

	// todo compare from current db data to old image data

	return true, nil
}

// todo
//func (b *BaseExecutor) queryCurrentRecords(conn *sql.Conn) *types.RecordImage {
//	tableMeta := b.undoImage.TableMeta
//	pkNameList := tableMeta.GetPrimaryKeyOnlyName()
//
//	b.undoImage.Rows
//
//}
//
//func (b *BaseExecutor) parsePkValues(rows []types.RowImage, pkNameList []string) {
//
//}
