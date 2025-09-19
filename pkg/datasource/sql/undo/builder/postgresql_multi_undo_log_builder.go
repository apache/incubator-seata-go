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
	"fmt"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

type PostgreSQLMultiUndoLogBuilder struct {
	BasicUndoLogBuilder
	beforeImages []*types.RecordImage
}

func GetPostgreSQLMultiUndoLogBuilder() undo.UndoLogBuilder {
	return &PostgreSQLMultiUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
		beforeImages:        make([]*types.RecordImage, 0),
	}
}

func (p *PostgreSQLMultiUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	if len(execCtx.ParseContext.MultiStmt) == 0 {
		return nil, nil
	}

	p.beforeImages = make([]*types.RecordImage, 0)
	resultImages := make([]*types.RecordImage, 0)
	
	originalExecCtx := execCtx

	for i, parseContext := range execCtx.ParseContext.MultiStmt {
		newExecCtx := &types.ExecContext{
			TxCtx:        originalExecCtx.TxCtx,
			ParseContext: parseContext,
			NamedValues:  originalExecCtx.NamedValues,
			Values:       originalExecCtx.Values,
			MetaDataMap:  originalExecCtx.MetaDataMap,
			Conn:         originalExecCtx.Conn,
			DBName:       originalExecCtx.DBName,
		}
		
		var tmpImages []*types.RecordImage
		var err error
		
		switch parseContext.ExecutorType {
		case types.UpdateExecutor:
			tmpImages, err = GetPostgreSQLMultiUpdateUndoLogBuilder().BeforeImage(ctx, newExecCtx)
		case types.DeleteExecutor:
			tmpImages, err = GetPostgreSQLMultiDeleteUndoLogBuilder().BeforeImage(ctx, newExecCtx)
		case types.InsertExecutor:
			tmpImages, err = GetPostgreSQLInsertUndoLogBuilder().BeforeImage(ctx, newExecCtx)
		case types.InsertOnDuplicateExecutor:
			tmpImages, err = GetPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder().BeforeImage(ctx, newExecCtx)
		default:
			return nil, fmt.Errorf("unsupported executor type %v for statement %d", parseContext.ExecutorType, i)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to build before image for statement %d (type %v): %v", i, parseContext.ExecutorType, err)
		}
		
		p.beforeImages = append(p.beforeImages, tmpImages...)
		resultImages = append(resultImages, tmpImages...)
	}

	return resultImages, nil
}

func (p *PostgreSQLMultiUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if len(execCtx.ParseContext.MultiStmt) == 0 {
		return nil, nil
	}

	if len(p.beforeImages) == 0 {
		return nil, fmt.Errorf("no before images found, BeforeImage must be called first")
	}

	resultImages := make([]*types.RecordImage, 0)
	originalExecCtx := execCtx
	
	beforeImageIndex := 0

	for i, parseContext := range execCtx.ParseContext.MultiStmt {
		newExecCtx := &types.ExecContext{
			TxCtx:        originalExecCtx.TxCtx,
			ParseContext: parseContext,
			NamedValues:  originalExecCtx.NamedValues,
			Values:       originalExecCtx.Values,
			MetaDataMap:  originalExecCtx.MetaDataMap,
			Conn:         originalExecCtx.Conn,
			DBName:       originalExecCtx.DBName,
		}

		var tmpImages []*types.RecordImage
		var err error
		var currentBeforeImages []*types.RecordImage
		
		var expectedImageCount int
		switch parseContext.ExecutorType {
		case types.UpdateExecutor, types.DeleteExecutor:
			expectedImageCount = 1
		case types.InsertExecutor:
			expectedImageCount = 0
		case types.InsertOnDuplicateExecutor:
			expectedImageCount = 1
		default:
			return nil, fmt.Errorf("unsupported executor type %v for statement %d", parseContext.ExecutorType, i)
		}
		
		if expectedImageCount > 0 {
			if beforeImageIndex >= len(p.beforeImages) {
				return nil, fmt.Errorf("insufficient before images for statement %d: expected at least %d, got %d total", i, beforeImageIndex+expectedImageCount, len(p.beforeImages))
			}
			currentBeforeImages = []*types.RecordImage{p.beforeImages[beforeImageIndex]}
			beforeImageIndex++
		} else {
			currentBeforeImages = []*types.RecordImage{}
		}

		switch parseContext.ExecutorType {
		case types.UpdateExecutor:
			tmpImages, err = GetPostgreSQLMultiUpdateUndoLogBuilder().AfterImage(ctx, newExecCtx, currentBeforeImages)
		case types.DeleteExecutor:
			tmpImages, err = GetPostgreSQLMultiDeleteUndoLogBuilder().AfterImage(ctx, newExecCtx, currentBeforeImages)
		case types.InsertExecutor:
			tmpImages, err = GetPostgreSQLInsertUndoLogBuilder().AfterImage(ctx, newExecCtx, currentBeforeImages)
		case types.InsertOnDuplicateExecutor:
			tmpImages, err = GetPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder().AfterImage(ctx, newExecCtx, currentBeforeImages)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to build after image for statement %d (type %v): %v", i, parseContext.ExecutorType, err)
		}
		
		resultImages = append(resultImages, tmpImages...)
	}

	return resultImages, nil
}

func (p *PostgreSQLMultiUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.MultiExecutor
}