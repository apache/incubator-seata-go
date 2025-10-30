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
	"strings"
	"testing"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestPostgreSQLMultiUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := GetPostgreSQLMultiUndoLogBuilder()
	executorType := builder.GetExecutorType()
	
	if executorType != types.MultiExecutor {
		t.Errorf("Expected MultiExecutor, got %v", executorType)
	}
}

func TestPostgreSQLMultiUndoLogBuilder_BeforeImage_EmptyMultiStmt(t *testing.T) {
	builder := &PostgreSQLMultiUndoLogBuilder{}
	
	execCtx := &types.ExecContext{
		ParseContext: &types.ParseContext{
			MultiStmt: []*types.ParseContext{},
		},
	}
	
	images, err := builder.BeforeImage(nil, execCtx)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if images != nil {
		t.Errorf("Expected nil images for empty MultiStmt, got %v", images)
	}
}

func TestPostgreSQLMultiUndoLogBuilder_AfterImage_EmptyMultiStmt(t *testing.T) {
	builder := &PostgreSQLMultiUndoLogBuilder{}
	
	execCtx := &types.ExecContext{
		ParseContext: &types.ParseContext{
			MultiStmt: []*types.ParseContext{},
		},
	}
	
	images, err := builder.AfterImage(nil, execCtx, nil)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if images != nil {
		t.Errorf("Expected nil images for empty MultiStmt, got %v", images)
	}
}

func TestPostgreSQLMultiUndoLogBuilder_Initialization(t *testing.T) {
	builder := GetPostgreSQLMultiUndoLogBuilder()
	
	if builder == nil {
		t.Error("Expected non-nil builder")
	}
	
	pgBuilder, ok := builder.(*PostgreSQLMultiUndoLogBuilder)
	if !ok {
		t.Error("Expected PostgreSQLMultiUndoLogBuilder type")
	}
	
	if pgBuilder.beforeImages == nil {
		t.Error("Expected non-nil beforeImages slice")
	}
	
	if len(pgBuilder.beforeImages) != 0 {
		t.Errorf("Expected empty beforeImages slice, got length %d", len(pgBuilder.beforeImages))
	}
}

func TestPostgreSQLMultiUndoLogBuilder_UnsupportedExecutorType(t *testing.T) {
	builder := &PostgreSQLMultiUndoLogBuilder{}
	
	execCtx := &types.ExecContext{
		ParseContext: &types.ParseContext{
			MultiStmt: []*types.ParseContext{
				{
					ExecutorType: types.ExecutorType(999),
				},
			},
		},
	}
	
	_, err := builder.BeforeImage(nil, execCtx)
	
	if err == nil {
		t.Error("Expected error for unsupported executor type")
	}
	
	if !strings.Contains(err.Error(), "unsupported executor type") {
		t.Errorf("Expected 'unsupported executor type' in error message, got: %v", err)
	}
}

func TestPostgreSQLMultiUndoLogBuilder_AfterImage_NoBeforeImages(t *testing.T) {
	builder := &PostgreSQLMultiUndoLogBuilder{}
	
	execCtx := &types.ExecContext{
		ParseContext: &types.ParseContext{
			MultiStmt: []*types.ParseContext{
				{
					ExecutorType: types.UpdateExecutor,
				},
			},
		},
	}
	
	_, err := builder.AfterImage(nil, execCtx, nil)
	
	if err == nil {
		t.Error("Expected error when no before images exist")
	}
	
	if !strings.Contains(err.Error(), "no before images found") {
		t.Errorf("Expected 'no before images found' in error message, got: %v", err)
	}
}

func TestPostgreSQLMultiUndoLogBuilder_AfterImage_UnsupportedExecutorType(t *testing.T) {
	builder := &PostgreSQLMultiUndoLogBuilder{
		beforeImages: []*types.RecordImage{{}},
	}
	
	execCtx := &types.ExecContext{
		ParseContext: &types.ParseContext{
			MultiStmt: []*types.ParseContext{
				{
					ExecutorType: types.ExecutorType(999),
				},
			},
		},
	}
	
	_, err := builder.AfterImage(nil, execCtx, nil)
	
	if err == nil {
		t.Error("Expected error for unsupported executor type in AfterImage")
	}
	
	if !strings.Contains(err.Error(), "unsupported executor type") {
		t.Errorf("Expected 'unsupported executor type' in error message, got: %v", err)
	}
}