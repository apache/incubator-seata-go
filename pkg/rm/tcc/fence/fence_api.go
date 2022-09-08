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

package fence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/seata/seata-go/pkg/common/errors"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/enum"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/handler"
	"github.com/seata/seata-go/pkg/tm"
)

// WithFence This method is a suspended API interface that asserts the phase timing of a transaction
// and performs corresponding database operations to ensure transaction consistency
// case 1: if fencePhase is FencePhaseNotExist, will return a fence not found error.
// case 2: if fencePhase is FencePhasePrepare, will do prepare fence operation.
// case 3: if fencePhase is FencePhaseCommit, will do commit fence operation.
// case 4: if fencePhase is FencePhaseRollback, will do rollback fence operation.
// case 5: if fencePhase not in above case, will return a fence phase illegal error.
func WithFence(ctx context.Context, tx *sql.Tx, callback func() error) (resErr error) {
	fencePhase := tm.GetFencePhase(ctx)

	// deal panic and return error
	defer func() {
		if rec := recover(); rec != nil {
			log.Error(rec)
			resErr = errors.NewTccFenceError(errors.FencePanicError,
				fmt.Sprintf("fence throw a panic, the msg is: %v", rec),
				nil,
			)
		}
	}()

	h := handler.GetFenceHandler()
	switch {
	case fencePhase == enum.FencePhaseNotExist:
		resErr = errors.NewTccFenceError(
			errors.FencePhaseError,
			fmt.Sprintf("xid %s, tx name %s, fence phase not exist", tm.GetXID(ctx), tm.GetTxName(ctx)),
			nil,
		)
	case fencePhase == enum.FencePhasePrepare:
		resErr = h.PrepareFence(ctx, tx, callback)
	case fencePhase == enum.FencePhaseCommit:
		resErr = h.CommitFence(ctx, tx, callback)
	case fencePhase == enum.FencePhaseRollback:
		resErr = h.RollbackFence(ctx, tx, callback)
	default:
		resErr = errors.NewTccFenceError(
			errors.FencePhaseError,
			fmt.Sprintf("fence phase: %v illegal", fencePhase),
			nil,
		)
	}

	if resErr != nil {
		log.Error(resErr)
	}

	return
}
