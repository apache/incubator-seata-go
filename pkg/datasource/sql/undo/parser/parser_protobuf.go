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

	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

type ProtobufParser struct {
}

// GetName get the name of parser
func (p *ProtobufParser) GetName() string {
	return "protobuf"
}

// GetDefaultContent get default content of this parser
func (p *ProtobufParser) GetDefaultContent() []byte {
	return []byte{}
}

// Encode branch undo log to byte array
func (p *ProtobufParser) Encode(branchUndoLog *undo.BranchUndoLog) ([]byte, error) {
	protoLog := ConvertToProto(branchUndoLog)
	return proto.Marshal(protoLog)
}

// Decode byte array to branch undo log
func (p *ProtobufParser) Decode(data []byte) (*undo.BranchUndoLog, error) {
	branchUndoLog := &BranchUndoLog{}
	err := proto.Unmarshal(data, branchUndoLog)
	if err != nil {
		return nil, err
	}

	return ConvertToIntree(branchUndoLog), nil
}

func ConvertToProto(intreeLog *undo.BranchUndoLog) *BranchUndoLog {
	protoLog := &BranchUndoLog{
		Xid:      intreeLog.Xid,
		BranchID: intreeLog.BranchID,
		Logs:     []*SQLUndoLog{},
	}
	for _, undolog := range intreeLog.Logs {
		protolog := &SQLUndoLog{
			SQLType:   SQLType(undolog.SQLType),
			TableName: undolog.TableName,
		}

		if undolog.BeforeImage != nil {
			protolog.BeforeImage = &RecordImage{
				TableName: undolog.BeforeImage.TableName,
				SQLType:   SQLType(undolog.BeforeImage.SQLType),
				Rows:      []*RowImage{},
			}

			for _, row := range undolog.BeforeImage.Rows {
				protoRow := &RowImage{
					Columns: []*ColumnImage{},
				}

				for _, col := range row.Columns {
					anyValue, err := convertInterfaceToAny(col.GetActualValue())
					if err != nil {
						continue
					}

					protoCol := &ColumnImage{
						KeyType:    IndexType(col.KeyType),
						ColumnName: col.ColumnName,
						ColumnType: JDBCType(col.ColumnType),
						Value:      anyValue,
					}

					protoRow.Columns = append(protoRow.Columns, protoCol)
				}

				protolog.BeforeImage.Rows = append(protolog.BeforeImage.Rows, protoRow)
			}
		}

		if undolog.AfterImage != nil {
			protolog.AfterImage = &RecordImage{
				TableName: undolog.AfterImage.TableName,
				SQLType:   SQLType(undolog.AfterImage.SQLType),
				Rows:      []*RowImage{},
			}

			for _, row := range undolog.AfterImage.Rows {
				protoRow := &RowImage{
					Columns: []*ColumnImage{},
				}

				for _, col := range row.Columns {
					anyValue, err := convertInterfaceToAny(col.Value)
					if err != nil {
						continue
					}

					protoCol := &ColumnImage{
						KeyType:    IndexType(col.KeyType),
						ColumnName: col.ColumnName,
						ColumnType: JDBCType(col.ColumnType),
						Value:      anyValue,
					}

					protoRow.Columns = append(protoRow.Columns, protoCol)
				}

				protolog.AfterImage.Rows = append(protolog.AfterImage.Rows, protoRow)
			}
		}

		protoLog.Logs = append(protoLog.Logs, protolog)
	}
	return protoLog
}

func ConvertToIntree(protoLog *BranchUndoLog) *undo.BranchUndoLog {
	intreeLog := &undo.BranchUndoLog{
		Xid:      protoLog.Xid,
		BranchID: protoLog.BranchID,
		Logs:     []undo.SQLUndoLog{},
	}

	for _, pbSqlLog := range protoLog.Logs {
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
					anyValue, err := convertAnyToInterface(pbCol.Value)
					if err != nil {
						continue
					}

					undoCol := types.ColumnImage{
						KeyType:    types.IndexType(pbCol.KeyType),
						ColumnName: pbCol.ColumnName,
						ColumnType: types.JDBCType(pbCol.ColumnType),
						Value:      anyValue,
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
					anyValue, err := convertAnyToInterface(pbCol.Value)
					if err != nil {
						continue
					}

					undoCol := types.ColumnImage{
						KeyType:    types.IndexType(pbCol.KeyType),
						ColumnName: pbCol.ColumnName,
						ColumnType: types.JDBCType(pbCol.ColumnType),
						Value:      anyValue,
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

func convertAnyToInterface(anyValue *any.Any) (interface{}, error) {
	var value interface{}
	bytesValue := &wrappers.BytesValue{}
	err := anypb.UnmarshalTo(anyValue, bytesValue, proto.UnmarshalOptions{})
	if err != nil {
		return value, err
	}
	uErr := json.Unmarshal(bytesValue.Value, &value)
	if uErr != nil {
		return value, uErr
	}
	return value, nil
}

func convertInterfaceToAny(v interface{}) (*any.Any, error) {
	anyValue := &any.Any{}
	bytes, _ := json.Marshal(v)
	bytesValue := &wrappers.BytesValue{
		Value: bytes,
	}
	err := anypb.MarshalFrom(anyValue, bytesValue, proto.MarshalOptions{})
	return anyValue, err
}
