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

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
)

func TestInitSeataContext(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	assert.NotNil(t, ctx.Value(seataContextVariable))
}

func TestSetTxStatus(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	SetTxStatus(ctx, message.GlobalStatusBegin)
	assert.Equal(t, message.GlobalStatusBegin,
		ctx.Value(seataContextVariable).(*ContextVariable).TxStatus)
}

func TestGetTxStatus(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	SetTxStatus(ctx, message.GlobalStatusBegin)
	assert.Equal(t, message.GlobalStatusBegin, *GetTxStatus(ctx))
}

func TestSetTxName(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	SetTxName(ctx, "GlobalTransaction")
	assert.Equal(t, "GlobalTransaction",
		ctx.Value(seataContextVariable).(*ContextVariable).TxName)
}

func TestGetTxName(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	SetTxName(ctx, "GlobalTransaction")
	assert.Equal(t, "GlobalTransaction", GetTxName(ctx))
}

func TestIsSeataContext(t *testing.T) {
	ctx := context.Background()
	assert.False(t, IsSeataContext(ctx))
	ctx = InitSeataContext(ctx)
	assert.True(t, IsSeataContext(ctx))
}

func TestSetBusinessActionContext(t *testing.T) {
	bac := &BusinessActionContext{}
	ctx := InitSeataContext(context.Background())
	SetBusinessActionContext(ctx, bac)
	assert.Equal(t, bac,
		ctx.Value(seataContextVariable).(*ContextVariable).BusinessActionContext)
}

func TestGetBusinessActionContext(t *testing.T) {
	bac := &BusinessActionContext{}
	ctx := InitSeataContext(context.Background())
	SetBusinessActionContext(ctx, bac)
	assert.Equal(t, bac, GetBusinessActionContext(ctx))
}

func TestSetTransactionRole(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	SetTxRole(ctx, Launcher)
	assert.Equal(t, Launcher,
		ctx.Value(seataContextVariable).(*ContextVariable).TxRole)
}

func TestGetTransactionRole(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	SetTxRole(ctx, Launcher)
	assert.Equal(t, Launcher,
		*GetTxRole(ctx))
}

func TestSetXID(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	xid := "12345"
	SetXID(ctx, xid)
	assert.Equal(t, xid,
		ctx.Value(seataContextVariable).(*ContextVariable).Xid)
}

func TestGetXID(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	xid := "12345"
	SetXID(ctx, xid)
	assert.Equal(t, xid,
		GetXID(ctx))
}

func TestIsGlobalTx(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	assert.False(t, IsGlobalTx(ctx))
	xid := "12345"
	SetXID(ctx, xid)
	assert.True(t, IsGlobalTx(ctx))
}

func TestSetXIDCopy(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	xid := "12345"
	SetXIDCopy(ctx, xid)
	assert.Equal(t, xid,
		ctx.Value(seataContextVariable).(*ContextVariable).XidCopy)
	assert.Equal(t, xid, GetXID(ctx))
}

func TestUnbindXid(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	xid := "12345"
	SetXID(ctx, xid)
	assert.Equal(t, xid, GetXID(ctx))
	UnbindXid(ctx)
	assert.Empty(t, GetXID(ctx))
}

func TestSetFencePhase(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	phase := enum.FencePhaseCommit
	SetFencePhase(ctx, phase)
	assert.Equal(t, phase,
		ctx.Value(seataContextVariable).(*ContextVariable).FencePhase)
}

func TestGetFencePhase(t *testing.T) {
	ctx := InitSeataContext(context.Background())
	phase := enum.FencePhaseCommit
	SetFencePhase(ctx, phase)
	assert.Equal(t, phase,
		GetFencePhase(ctx))
}
