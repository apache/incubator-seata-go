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
	"encoding/json"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
)

type ProtostuffParser struct {
}

// Get the name of parser;
// return the name of parser
func (p *ProtostuffParser) GetName() string {
	return "protostuff"
}

// Get default context of this parser
// return the default content if undo log is empty
func (p *ProtostuffParser) GetDefaultContent() []byte {
	return []byte("{}")
}

// Encode branch undo log to byte array.
// param branchUndoLog the branch undo log
// return the byte array
func (p *ProtostuffParser) Encode(branchUndoLog *undo.BranchUndoLog) ([]byte, error) {
	bytes, err := json.Marshal(branchUndoLog)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// Decode byte array to branch undo log.
// param bytes the byte array
// return the branch undo log
func (p *ProtostuffParser) Decode(bytes []byte) (*undo.BranchUndoLog, error) {
	var branchUndoLog *undo.BranchUndoLog
	if err := json.Unmarshal(bytes, &branchUndoLog); err != nil {
		return nil, err
	}

	return branchUndoLog, nil
}

func ConvertPbBranchUndoLogToIntreeBranchUndoLog(pbLog *BranchUndoLog) *undo.BranchUndoLog {
	intreeLog := &undo.BranchUndoLog{
		Xid:      pbLog.Xid,
		BranchID: pbLog.BranchID,
		Logs:     []undo.SQLUndoLog{},
	}

	for _, pbSqlLog := range pbLog.Logs {
		undoSqlLog := undo.SQLUndoLog{
			SQLType:   types.SQLType(pbSqlLog.SQLType),
			TableName: pbSqlLog.TableName,
		}

		if pbSqlLog.BeforeImage != nil {
			undoSqlLog.BeforeImage = &types.RecordImage{
				TableName: pbSqlLog.BeforeImage.TableName,
				SQLType:   types.SQLType(pbSqlLog.BeforeImage.SQLType),
				Rows:      []types.RowImage{},
			}

			for _, pbRow := range pbSqlLog.BeforeImage.Rows {
				undoRow := types.RowImage{
					Columns: []types.ColumnImage{},
				}

				for _, pbCol := range pbRow.Columns {
					m := pbCol.AsMap()
					undoCol := types.ColumnImage{
						KeyType:    types.IndexType(m["KeyType"].(IndexType)),
						ColumnName: m["ColumnName"].(string),
						ColumnType: types.JDBCType(m["ColumnType"].(JDBCType)),
						Value:      m["Value"],
					}

					undoRow.Columns = append(undoRow.Columns, undoCol)
				}

				undoSqlLog.BeforeImage.Rows = append(undoSqlLog.BeforeImage.Rows, undoRow)
			}
		}

		if pbSqlLog.AfterImage != nil {
			undoSqlLog.AfterImage = &types.RecordImage{
				TableName: pbSqlLog.AfterImage.TableName,
				SQLType:   types.SQLType(pbSqlLog.AfterImage.SQLType),
				Rows:      []types.RowImage{},
			}

			for _, pbRow := range pbSqlLog.AfterImage.Rows {
				undoRow := types.RowImage{
					Columns: []types.ColumnImage{},
				}

				for _, pbCol := range pbRow.Columns {
					m := pbCol.AsMap()
					undoCol := types.ColumnImage{
						KeyType:    types.IndexType(m["KeyType"].(float64)),
						ColumnName: m["ColumnName"].(string),
						ColumnType: types.JDBCType(m["ColumnType"].(JDBCType)),
						Value:      m["Value"],
					}

					undoRow.Columns = append(undoRow.Columns, undoCol)
				}

				undoSqlLog.AfterImage.Rows = append(undoSqlLog.AfterImage.Rows, undoRow)
			}
		}

		intreeLog.Logs = append(intreeLog.Logs, undoSqlLog)
	}

	return intreeLog
}

func ConvertIntreeBranchUndoLogToPbBranchUndoLog() {

}
