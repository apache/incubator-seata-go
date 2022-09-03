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

	"github.com/zouyx/agollo/v3/component/log"

	"github.com/seata/seata-go/pkg/common/errors"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/handler"
	"github.com/seata/seata-go/pkg/tm"
)

func WithFence(ctx context.Context, tx *sql.Tx, callback func() error) (resErr error) {
	fencePhase := tm.GetFencePhase(ctx)
	if fencePhase == constant.FencePhaseNotExist {
		return fmt.Errorf("xid %s, tx name %s, fence phase not exist", tm.GetXID(ctx), tm.GetTxName(ctx))
	}

	// deal panic and return error
	defer func() {
		rec := recover()
		if rec != nil {
			log.Error(rec)
			resErr = errors.NewTccFenceError(errors.FencePanicError,
				fmt.Sprintf("fence throw a panic, the msg is: %v", rec),
				nil,
			)
		}
	}()

	if fencePhase == constant.FencePhasePrepare {
		resErr = handler.GetFenceHandlerSingleton().PrepareFence(ctx, tx, callback)
	} else if fencePhase == constant.FencePhaseCommit {
		resErr = handler.GetFenceHandlerSingleton().CommitFence(ctx, tx, callback)
	} else if fencePhase == constant.FencePhaseRollback {
		resErr = handler.GetFenceHandlerSingleton().RollbackFence(ctx, tx, callback)
	} else {
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
