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

type UpdateExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContent *types.ExecContext
}

func NewUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext) *UpdateExecutor {
	return &UpdateExecutor{parserCtx: parserCtx, execContent: execContent}
}

func (u *UpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	beforeImage, err := u.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, u.execContent.Query, u.execContent.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := u.afterImage(ctx, beforeImage)
	if err != nil {
		return nil, err
	}

	if len(beforeImage) != len(afterImage) {
		return nil, fmt.Errorf("Before image size is not equaled to after image size, probably because you updated the primary keys.")
	}

	u.execContent.TxCtx.RoundImages.AppendAfterImages(beforeImage)
	u.execContent.TxCtx.RoundImages.AppendAfterImages(afterImage)

	return res, nil
}

func (u *UpdateExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	return nil, nil
}

func (u *UpdateExecutor) afterImage(ctx context.Context, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	return nil, nil
}
