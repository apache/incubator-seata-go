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
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm/tcc/api"
)

type ContextVariable struct {
	TxName                string
	Xid                   string
	Status                *message.GlobalStatus
	TxRole                *GlobalTransactionRole
	BusinessActionContext *api.BusinessActionContext
	TxStatus              *message.GlobalStatus
}

func InitSeataContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, common.CONTEXT_VARIABLE, &ContextVariable{})
}

func GetTxStatus(ctx context.Context) *message.GlobalStatus {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return nil
	}
	return variable.(*ContextVariable).TxStatus
}

func SetTxStatus(ctx context.Context, status message.GlobalStatus) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).TxStatus = &status
	}
}

func GetTxName(ctx context.Context) string {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return ""
	}
	return variable.(*ContextVariable).TxName
}

func SetTxName(ctx context.Context, name string) {
	variable := ctx.Value(common.TccBusinessActionContext)
	if variable != nil {
		variable.(*ContextVariable).TxName = name
	}
}

func IsSeataContext(ctx context.Context) bool {
	return ctx.Value(common.CONTEXT_VARIABLE) != nil
}

func GetBusinessActionContext(ctx context.Context) *api.BusinessActionContext {
	variable := ctx.Value(common.TccBusinessActionContext)
	if variable == nil {
		return nil
	}
	return variable.(*api.BusinessActionContext)
}

func SetBusinessActionContext(ctx context.Context, businessActionContext *api.BusinessActionContext) {
	variable := ctx.Value(common.TccBusinessActionContext)
	if variable != nil {
		variable.(*ContextVariable).BusinessActionContext = businessActionContext
	}
}

func GetTransactionRole(ctx context.Context) *GlobalTransactionRole {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return nil
	}
	return variable.(*ContextVariable).TxRole
}

func SetTransactionRole(ctx context.Context, role GlobalTransactionRole) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).TxRole = &role
	}
}

func GetXID(ctx context.Context) string {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return ""
	}
	return variable.(*ContextVariable).Xid
}

func HasXID(ctx context.Context) bool {
	return GetXID(ctx) != ""
}

func SetXID(ctx context.Context, xid string) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).Xid = xid
	}
}

func UnbindXid(ctx context.Context) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).Xid = ""
	}
}
