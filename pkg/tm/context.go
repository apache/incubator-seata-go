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

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
)

type ContextParam string

const (
	seataContextVariable = ContextParam("seataContextVariable")
)

type GlobalTransaction struct {
	Xid     string
	XidCopy string
	TxName  string
	// TxStatus Identify a global transaction in a certain status
	TxStatus message.GlobalStatus
	// TxRole Roles in the transaction propagation behavior
	TxRole GlobalTransactionRole
}

type BusinessActionContext struct {
	Xid           string
	BranchId      int64
	ActionName    string
	IsDelayReport bool
	IsUpdated     bool
	ActionContext map[string]interface{}
}

type ContextVariable struct {
	FencePhase     enum.FencePhase
	FenceTxBegined bool

	BusinessActionContext *BusinessActionContext
	// GlobalTransaction Represent seata ctx is a global transaction
	GlobalTransaction
}

func InitSeataContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, seataContextVariable, &ContextVariable{})
}

func GetTxStatus(ctx context.Context) *message.GlobalStatus {
	variable := ctx.Value(seataContextVariable)
	if variable == nil {
		return nil
	}
	return &variable.(*ContextVariable).TxStatus
}

func SetTxStatus(ctx context.Context, status message.GlobalStatus) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).TxStatus = status
	}
}

func GetTxName(ctx context.Context) string {
	variable := ctx.Value(seataContextVariable)
	if variable == nil {
		return ""
	}
	return variable.(*ContextVariable).TxName
}

func SetTxName(ctx context.Context, name string) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).TxName = name
	}
}

func IsSeataContext(ctx context.Context) bool {
	return ctx.Value(seataContextVariable) != nil
}

func GetBusinessActionContext(ctx context.Context) *BusinessActionContext {
	variable := ctx.Value(seataContextVariable)
	if variable == nil {
		return nil
	}
	return variable.(*ContextVariable).BusinessActionContext
}

func SetBusinessActionContext(ctx context.Context, businessActionContext *BusinessActionContext) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).BusinessActionContext = businessActionContext
	}
}

func GetTxRole(ctx context.Context) *GlobalTransactionRole {
	variable := ctx.Value(seataContextVariable)
	if variable == nil {
		return nil
	}
	return &variable.(*ContextVariable).TxRole
}

func SetTxRole(ctx context.Context, role GlobalTransactionRole) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).TxRole = role
	}
}

func IsGlobalTx(ctx context.Context) bool {
	variable := ctx.Value(seataContextVariable)
	if variable == nil {
		return false
	}
	return variable.(*ContextVariable).Xid != ""
}

func GetXID(ctx context.Context) string {
	variable := ctx.Value(seataContextVariable)
	if variable == nil {
		return ""
	}
	xid := variable.(*ContextVariable).Xid
	if xid == "" {
		xid = variable.(*ContextVariable).XidCopy
	}
	return xid
}

func SetXID(ctx context.Context, xid string) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).Xid = xid
	}
}

func SetXIDCopy(ctx context.Context, xid string) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).XidCopy = xid
	}
}

func UnbindXid(ctx context.Context) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).Xid = ""
		variable.(*ContextVariable).XidCopy = ""
	}
}

func SetTx(ctx context.Context, tx *GlobalTransaction) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).GlobalTransaction = *tx
	}
}

func GetTx(ctx context.Context) (tx *GlobalTransaction) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		tx = &variable.(*ContextVariable).GlobalTransaction
	}
	return
}

func SetFencePhase(ctx context.Context, phase enum.FencePhase) {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		variable.(*ContextVariable).FencePhase = phase
	}
}

func GetFencePhase(ctx context.Context) enum.FencePhase {
	variable := ctx.Value(seataContextVariable)
	if variable != nil {
		return variable.(*ContextVariable).FencePhase
	}
	return enum.FencePhaseNotExist
}

func SetFenceTxBeginedFlag(ctx context.Context, fenceTxBegined bool) {
	if variable := ctx.Value(seataContextVariable); variable != nil {
		variable.(*ContextVariable).FenceTxBegined = fenceTxBegined
	}
}

func IsFenceTxBegin(ctx context.Context) bool {
	if variable := ctx.Value(seataContextVariable); variable != nil {
		return variable.(*ContextVariable).FenceTxBegined
	}

	return false
}
