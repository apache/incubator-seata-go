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

package exec

import (
	"context"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

// UpdateUndoHook build image when update executed
type UpdateUndoHook struct {
	basicUndoBuilder BasicUndoBuilder
}

func (u UpdateUndoHook) Type() types.SQLType {
	return types.SQLTypeUpdate
}

// Before build image before execute business sql
func (u UpdateUndoHook) Before(ctx context.Context, execCtx *exec.ExecContext) {
	r, err := u.basicUndoBuilder.buildRecordImage(ctx, execCtx)
	if err != nil {
		log.Fatalf("build before iamge %v", err)
		return
	}
	execCtx.TxCtx.RoundImages.AppendBeofreImages(r)
}

// Before build image after execute business sql
func (u UpdateUndoHook) After(ctx context.Context, execCtx *exec.ExecContext) {
	r, err := u.basicUndoBuilder.buildRecordImage(ctx, execCtx)
	if err != nil {
		log.Fatalf("build after iamge %v", err)
		return
	}
	execCtx.TxCtx.RoundImages.AppendAfterImages(r)
}
