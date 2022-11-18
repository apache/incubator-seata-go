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
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/protocol/message"
)

func TestTransactionExecutorBegin(t *testing.T) {
	type Test struct {
		ctx                context.Context
		name               string
		xid                string
		wantHasError       bool
		wantErrString      string
		wantHasMock        bool
		wantMockTargetName string
		wantMockFunction   interface{}
	}
	gts := []Test{
		// this case to test Begin basic path
		{
			ctx:           context.Background(),
			name:          "zhangsan",
			xid:           "123456",
			wantHasError:  true,
			wantErrString: "Ignore GlobalStatusBegin(): just involved in global transaction 123456",
		},
		// this case to test the TransactionManager#Begin return path
		{
			ctx:                context.Background(),
			name:               "zhangsan",
			xid:                "123456",
			wantHasError:       true,
			wantErrString:      "transactionTemplate: begin transaction failed, error mock transaction executor begin",
			wantHasMock:        true,
			wantMockTargetName: "Begin",
			wantMockFunction: func(_ *GlobalTransactionManager, ctx context.Context, tx *GlobalTransaction, i time.Duration, s string) error {
				return errors.New("mock transaction executor begin")
			},
		},
	}

	for _, v := range gts {
		var stub *gomonkey.Patches
		// set up stub
		if v.wantHasMock {
			stub = gomonkey.ApplyMethod(reflect.TypeOf(GetGlobalTransactionManager()), v.wantMockTargetName, v.wantMockFunction)
		}

		if v.xid != "" {
			v.ctx = InitSeataContext(v.ctx)
			SetXID(v.ctx, v.xid)
		}

		func() {
			// Begin will throw the panic, so there need to recover it and assert.
			defer func(v Test) {
				err, ok := recover().(error)
				if ok && err != nil && v.wantHasError {
					assert.Equal(t, v.wantErrString, err.Error)
				}
			}(v)
			begin(v.ctx, &TransactionInfo{
				Name: v.name,
			})
		}()

		// rest up stub
		if v.wantHasMock {
			stub.Reset()
		}
	}
}

func TestTransactionExecutorCommit(t *testing.T) {
	ctx := context.Background()
	ctx = InitSeataContext(ctx)
	SetTransactionRole(ctx, LAUNCHER)
	SetTxStatus(ctx, message.GlobalStatusBegin)
	SetXID(ctx, "")
	assert.Equal(t, "Commit xid should not be empty", commitOrRollback(ctx, true).Error())
}

func TestTransactionExecurotRollback(t *testing.T) {
	ctx := context.Background()
	ctx = InitSeataContext(ctx)
	SetTransactionRole(ctx, LAUNCHER)
	SetTxStatus(ctx, message.GlobalStatusBegin)
	SetXID(ctx, "")
	errActual := commitOrRollback(ctx, false)
	assert.Equal(t, "Rollback xid should not be empty", errActual.Error())
}
