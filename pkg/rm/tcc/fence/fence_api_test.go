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
	"testing"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestWithFence(t *testing.T) {
	log.Init()
	tests := []struct {
		xid          string
		txName       string
		fencePhase   enum.FencePhase
		bac          tm.BusinessActionContext
		callback     func() error
		wantErr      bool
		errStr       string
		wantCommit   bool
		wantRollback bool
	}{
		{
			xid:    "123",
			txName: "test",
			callback: func() error {
				return nil
			},
			wantErr: true,
			errStr:  "xid 123, tx name test, fence phase not exist",
		},
	}
	for _, v := range tests {
		db, mock, _ := sqlmock.New()
		mock.ExpectBegin()
		if v.wantCommit {
			mock.ExpectCommit()
		}
		if v.wantRollback {
			mock.ExpectRollback()
		}
		ctx := context.Background()
		ctx = tm.InitSeataContext(ctx)
		tm.SetXID(ctx, v.xid)
		tm.SetTxName(ctx, v.txName)
		tm.SetFencePhase(ctx, v.fencePhase)
		tm.SetBusinessActionContext(ctx, &v.bac)
		tx, _ := db.BeginTx(ctx, &sql.TxOptions{})

		if v.wantErr {
			assert.Equal(t, v.errStr, WithFence(ctx, tx, v.callback).Error())
		} else {
			assert.Nil(t, WithFence(ctx, tx, v.callback))
		}
	}
}
