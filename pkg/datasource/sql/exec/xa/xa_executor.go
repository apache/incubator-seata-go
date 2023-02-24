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

package xa

import (
	"context"
	"errors"

	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/util"
	"github.com/seata/seata-go/pkg/datasource/sql/xa"
	"github.com/seata/seata-go/pkg/util/log"
)

func Init() {
	exec.RegisterXAExecutor(types.DBTypeMySQL, func() exec.SQLExecutor {
		return &XAExecutor{}
	})
}

// XAExecutor The XA transaction manager.
type XAExecutor struct {
	proxy *xa.ConnectionProxyXA
}

// Interceptors set xa executor hooks
func (e *XAExecutor) Interceptors(hooks []exec.SQLHook) {
}

func (e *XAExecutor) SetConnectionProxyXA(proxy *xa.ConnectionProxyXA) {
	e.proxy = proxy
}

// ExecWithNamedValue exec first phase xa transaction, like(XA Start, Business SQL, XA End, Prepare)
func (e *XAExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if execCtx == nil {
		return nil, errors.New("xa executor exec with the nil exec context")
	}
	if e.proxy == nil {
		return nil, errors.New("xa executor exec with the nil xa connection proxy")
	}

	autoCommitStatus := e.proxy.GetAutoCommit()
	// XA Start
	if autoCommitStatus {
		e.proxy.SetAutoCommit(ctx, false)
	}

	defer func() {
		e.proxy.SetAutoCommit(ctx, true)
	}()

	// execute SQL
	res, err := f(ctx, execCtx.Query, execCtx.NamedValues)
	if err != nil {
		log.Errorf("xa executor do callback failure, sql:%s, xid:%s, err:%v", execCtx.Query, execCtx.TxCtx.XID, err)
		// XA End & Rollback
		if autoCommitStatus {
			if err := e.proxy.Rollback(ctx); err != nil {
				log.Errorf("xa connection proxy rollback failure sql:%s, xid:%s, err:%v", execCtx.Query, execCtx.TxCtx.XID, err)
			}
		}
		return nil, err
	}

	if autoCommitStatus {
		// XA End & Prepare
		if err := e.proxy.Commit(ctx); err != nil {
			log.Errorf("xa connection proxy commit failure sql:%s, xid:%s, err:%v", execCtx.Query, execCtx.TxCtx.XID, err)
			// XA End & Rollback
			if err := e.proxy.Rollback(ctx); err != nil {
				log.Errorf("xa connection proxy rollback failure sql:%s, xid:%s, err:%v", execCtx.Query, execCtx.TxCtx.XID, err)
			}
		}
	}

	return res, nil
}

func (e *XAExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	execCtx.NamedValues = util.ValueToNamedValue(execCtx.Values)
	return e.ExecWithNamedValue(ctx, execCtx, f)
}
