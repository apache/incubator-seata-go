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

package at

import (
	"context"
	"fmt"

	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

type multiExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

// NewMultiExecutor get new multi executor
func NewMultiExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) executor {
	return &multiExecutor{parserCtx: parserCtx, execContext: execContext, baseExecutor: baseExecutor{hooks: hooks}}
}

// ExecContext exec SQL, and generate before image and after image
func (m *multiExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	m.beforeHooks(ctx, m.execContext)

	defer func() {
		m.afterHooks(ctx, m.execContext)
	}()

	beforeImages, err := m.beforeImage(ctx, m.parserCtx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, m.execContext.Query, m.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImages, err := m.afterImage(ctx, m.parserCtx, beforeImages)
	if err != nil {
		return nil, err
	}

	for _, beforeImage := range beforeImages {
		m.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	}
	for _, afterImage := range afterImages {
		m.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	}

	return res, nil
}
func (m *multiExecutor) beforeImage(ctx context.Context, parseContext *types.ParseContext) ([]*types.RecordImage, error) {
	if len(parseContext.MultiStmt) == 0 {
		return nil, nil
	}
	var tmpImages []*types.RecordImage
	var err error

	var beforeImages = make([]*types.RecordImage, 0)
	for _, multiStmt := range parseContext.MultiStmt {
		switch multiStmt.ExecutorType {
		case types.UpdateExecutor:
			multiUpdateExec := NewMultiUpdateExecutor(m.parserCtx, m.execContext, m.hooks)
			tmpImages, err = multiUpdateExec.beforeImage(ctx)
		case types.DeleteExecutor:
			multiDeleteExec := NewMultiDeleteExecutor(m.parserCtx, m.execContext, m.hooks)
			tmpImages, err = multiDeleteExec.beforeImage(ctx)
		default:
			return nil, fmt.Errorf("not support sql %s", m.execContext.Query)
		}

		if err != nil {
			return nil, err
		}
		beforeImages = append(beforeImages, tmpImages...)
	}

	return beforeImages, err
}

func (m *multiExecutor) afterImage(ctx context.Context, parseContext *types.ParseContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if len(parseContext.MultiStmt) == 0 {
		return nil, nil
	}
	tmpImages := make([]*types.RecordImage, 0)
	var err error

	var afterImages = make([]*types.RecordImage, 0)
	for _, multiStmt := range parseContext.MultiStmt {
		switch multiStmt.ExecutorType {
		case types.UpdateExecutor:
			multiUpdateExec := NewMultiUpdateExecutor(m.parserCtx, m.execContext, m.hooks)
			tmpImages, err = multiUpdateExec.afterImage(ctx, beforeImages)
		case types.DeleteExecutor:
			// todo use MultiDeleteExecutor
		}
		afterImages = append(afterImages, tmpImages...)
	}
	if err != nil {
		return nil, err
	}
	return afterImages, err
}
