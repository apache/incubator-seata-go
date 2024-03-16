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

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/handler"
	"seata.apache.org/seata-go/pkg/tm"
)

// WithFence Execute the fence database operation first and then call back the business method
func WithFence(ctx context.Context, tx *sql.Tx, callback func() error) (err error) {
	if err = DoFence(ctx, tx); err != nil {
		return err
	}

	if err := callback(); err != nil {
		return fmt.Errorf("the business method error msg of: %p, [%w]", callback, err)
	}

	return
}

// DeFence This method is a suspended API interface that asserts the phase timing of a transaction
// and performs corresponding database operations to ensure transaction consistency
// case 1: if fencePhase is FencePhaseNotExist, will return a fence not found error.
// case 2: if fencePhase is FencePhasePrepare, will do prepare fence operation.
// case 3: if fencePhase is FencePhaseCommit, will do commit fence operation.
// case 4: if fencePhase is FencePhaseRollback, will do rollback fence operation.
// case 5: if fencePhase not in above case, will return a fence phase illegal error.
func DoFence(ctx context.Context, tx *sql.Tx) error {
	hd := handler.GetFenceHandler()
	phase := tm.GetFencePhase(ctx)

	switch phase {
	case enum.FencePhaseNotExist:
		return fmt.Errorf("xid %s, tx name %s, fence phase not exist", tm.GetXID(ctx), tm.GetTxName(ctx))
	case enum.FencePhasePrepare:
		return hd.PrepareFence(ctx, tx)
	case enum.FencePhaseCommit:
		return hd.CommitFence(ctx, tx)
	case enum.FencePhaseRollback:
		return hd.RollbackFence(ctx, tx)
	}

	return fmt.Errorf("fence phase: %v illegal", phase)
}
