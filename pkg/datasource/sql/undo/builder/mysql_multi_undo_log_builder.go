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

package builder

import (
	"context"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

type MySQLMultiUndoLogBuilder struct {
	BasicUndoLogBuilder
	beforeImages []*types.RecordImage
}

func GetMySQLMultiUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLMultiUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
		beforeImages:        make([]*types.RecordImage, 0),
	}
}

func (u *MySQLMultiUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	if len(execCtx.ParseContext.MultiStmt) == 0 {
		return nil, nil
	}

	var (
		err          error
		tmpImages    []*types.RecordImage
		resultImages []*types.RecordImage
	)

	resultImages = make([]*types.RecordImage, 0)
	for _, parseContext := range execCtx.ParseContext.MultiStmt {
		execCtx = &types.ExecContext{
			TxCtx:        execCtx.TxCtx,
			ParseContext: parseContext,
			NamedValues:  execCtx.NamedValues,
			Values:       execCtx.Values,
			MetaDataMap:  execCtx.MetaDataMap,
			Conn:         execCtx.Conn,
		}
		switch parseContext.ExecutorType {
		case types.UpdateExecutor:
			// todo change to use MultiUpdateExecutor
			tmpImages, err = GetMySQLUpdateUndoLogBuilder().BeforeImage(ctx, execCtx)
			break
		case types.DeleteExecutor:
			// todo use MultiDeleteExecutor
			tmpImages, err = GetMySQLMultiDeleteUndoLogBuilder().BeforeImage(ctx, execCtx)
			break
		}

		if err != nil {
			return nil, err
		}
		resultImages = append(resultImages, tmpImages...)
		u.beforeImages = append(u.beforeImages, tmpImages...)
	}

	return resultImages, err
}

func (u *MySQLMultiUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if len(execCtx.ParseContext.MultiStmt) == 0 {
		return nil, nil
	}

	var (
		err          error
		tmpImages    []*types.RecordImage
		resultImages = make([]*types.RecordImage, 0)
	)

	for i, parseContext := range execCtx.ParseContext.MultiStmt {
		execCtx = &types.ExecContext{
			TxCtx:        execCtx.TxCtx,
			ParseContext: parseContext,
			NamedValues:  execCtx.NamedValues,
			Values:       execCtx.Values,
			MetaDataMap:  execCtx.MetaDataMap,
			Conn:         execCtx.Conn,
		}

		switch parseContext.ExecutorType {
		case types.UpdateExecutor:
			tmpImages, err = GetMySQLMultiUpdateUndoLogBuilder().AfterImage(ctx, execCtx, []*types.RecordImage{u.beforeImages[i]})
			break
		case types.DeleteExecutor:
			// todo use MultiDeleteExecutor
			break
		}
		if err != nil {
			return nil, err
		}
		resultImages = append(resultImages, tmpImages...)
	}

	return resultImages, err
}

func (u *MySQLMultiUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.MultiExecutor
}
