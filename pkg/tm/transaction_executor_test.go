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

package tm

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/protocol/message"

	"github.com/stretchr/testify/assert"
)

func TestTransactionExecutorBegin(t *testing.T) {
	type Test struct {
		ctx           context.Context
		name          string
		xid           string
		wantHasError  bool
		wantErrString string
	}
	gts := []Test{
		{
			ctx:           context.Background(),
			name:          "zhangsan",
			xid:           "123456",
			wantHasError:  true,
			wantErrString: "Ignore GlobalStatusBegin(): just involved in global transaction 123456",
		},
		// todo other case depend on network service,
	}

	for _, v := range gts {
		if v.xid != "" {
			v.ctx = InitSeataContext(v.ctx)
			SetXID(v.ctx, v.xid)
		}
		Begin(v.ctx, v.name)
		defer func(v Test) {
			err, ok := recover().(error)
			if ok && err != nil && v.wantHasError {
				assert.Equal(t, v.wantErrString, err.Error)
			}
		}(v)
	}
}

func TestTransactionExecutorCommit(t *testing.T) {
	ctx := context.Background()
	ctx = InitSeataContext(ctx)
	SetTransactionRole(ctx, LAUNCHER)
	SetTxStatus(ctx, message.GlobalStatusBegin)
	SetXID(ctx, "")
	var err error = nil
	assert.Equal(t, "Commit xid should not be empty", CommitOrRollback(ctx, &err).Error())
}

func TestTransactionExecurotRollback(t *testing.T) {
	ctx := context.Background()
	ctx = InitSeataContext(ctx)
	SetTransactionRole(ctx, LAUNCHER)
	SetTxStatus(ctx, message.GlobalStatusBegin)
	SetXID(ctx, "")
	errExpect := errors.New("rollback error")
	errActual := CommitOrRollback(ctx, &errExpect)
	assert.Equal(t, "Rollback xid should not be empty", errActual.Error())
}
