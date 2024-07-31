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

package dubbo

import (
	"context"
	"strings"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	seataFilter *dubboTransactionFilter
	once        sync.Once
)

func InitSeataDubbo() {
	extension.SetFilter(constant.SeataFilterKey, GetDubboTransactionFilter)
}

type Filter interface{}

type dubboTransactionFilter struct{}

func GetDubboTransactionFilter() filter.Filter {
	if seataFilter == nil {
		once.Do(func() {
			seataFilter = &dubboTransactionFilter{}
		})
	}
	return seataFilter
}

func (d *dubboTransactionFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	xid := tm.GetXID(ctx)
	rpcXid := d.getRpcXid(invocation)
	log.Infof("xid in context is %s, xid in RpcContextis %s", xid, rpcXid)

	if xid != "" {
		// dubbo go
		invocation.SetAttachment(constant.SeataXidKey, xid)
		// dubbo java
		invocation.SetAttachment(constant.XidKey, xid)
	} else if rpcXid != xid {
		ctx = tm.InitSeataContext(ctx)
		tm.SetXID(ctx, rpcXid)
	}
	return invoker.Invoke(ctx, invocation)
	// todo why should unbind xid???
}

func (*dubboTransactionFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func (d *dubboTransactionFilter) getRpcXid(invocation protocol.Invocation) string {
	rpcXid := d.getDubboGoRpcXid(invocation)
	if rpcXid == "" {
		rpcXid = d.getDubboJavaRpcXid(invocation)
	}
	return rpcXid
}

func (*dubboTransactionFilter) getDubboGoRpcXid(invocation protocol.Invocation) string {
	rpcXid := invocation.GetAttachmentWithDefaultValue(constant.SeataXidKey, "")
	if rpcXid == "" {
		rpcXid = invocation.GetAttachmentWithDefaultValue(strings.ToLower(constant.SeataXidKey), "")
	}
	return rpcXid
}

func (*dubboTransactionFilter) getDubboJavaRpcXid(invocation protocol.Invocation) string {
	rpcXid := invocation.GetAttachmentWithDefaultValue(constant.XidKey, "")
	if rpcXid == "" {
		rpcXid = invocation.GetAttachmentWithDefaultValue(strings.ToLower(constant.XidKey), "")
	}
	return rpcXid
}
