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

package hook

import (
	"context"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/tm"
)

func NewUndoLogSQLHook() exec.SQLHook {
	return &undoLogSQLHook{}
}

type undoLogSQLHook struct {
}

func (h *undoLogSQLHook) Type() types.SQLType {
	return types.SQLTypeUnknown
}

// Before
func (h *undoLogSQLHook) Before(ctx context.Context, execCtx *types.ExecContext) error {
	if !tm.IsGlobalTx(ctx) {
		return nil
	}

	pc, err := parser.DoParser(execCtx.Query)
	if err != nil {
		return err
	}
	if !pc.HasValidStmt() {
		return nil
	}

	builder := undo.GetUndologBuilder(pc.ExecutorType)
	if builder == nil {
		return nil
	}
	recordImage, err := builder.BeforeImage(ctx, execCtx)
	if err != nil {
		return err
	}
	execCtx.TxCtx.RoundImages.AppendBeofreImages(recordImage)
	return nil
}

// After
func (h *undoLogSQLHook) After(ctx context.Context, execCtx *types.ExecContext) error {
	if !tm.IsGlobalTx(ctx) {
		return nil
	}
	return nil
}
