package dubbo

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/tm"
	"strings"
	"sync"
)

var (
	seataFilter *DubboTransactionFilter
	once        sync.Once
)

// register dubbo filter
func init() {
	extension.SetFilter(common.SeataFilterKey, newDubboTransactionFilter)
}

type Filter interface {
}

type DubboTransactionFilter struct {
}

func newDubboTransactionFilter() filter.Filter {
	if seataFilter == nil {
		once.Do(func() {
			seataFilter = &DubboTransactionFilter{}
		})
	}
	return seataFilter
}

func (d *DubboTransactionFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	// context should be inited as seata context
	if !tm.IsSeataContext(ctx) {
		return invoker.Invoke(ctx, invocation)
	}
	xid := tm.GetXID(ctx)
	branchType := tm.GetBranchType(ctx)
	rpcXid := d.getRpcXid(invocation)
	log.Infof("xid in context is %s, branch type is %v, xid in RpcContextis %s", xid, branchType, rpcXid)

	if xid != "" {
		// dubbo go
		invocation.SetAttachment(common.SeataXidKey, xid)
		// dubbo java
		invocation.SetAttachment(common.XidKey, xid)
		invocation.SetAttachment(common.BranchTypeKey, branchType)
	} else if rpcXid != xid {
		tm.SetXID(ctx, rpcXid)
	}
	return invoker.Invoke(ctx, invocation)
	// todo why should unbind xid???
}

func (*DubboTransactionFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func (d *DubboTransactionFilter) getRpcXid(invocation protocol.Invocation) string {
	rpcXid := d.getDubboGoRpcXid(invocation)
	if rpcXid == "" {
		rpcXid = d.getDubboJavaRpcXid(invocation)
	}
	return rpcXid
}

func (*DubboTransactionFilter) getDubboGoRpcXid(invocation protocol.Invocation) string {
	rpcXid := invocation.GetAttachmentWithDefaultValue(common.SeataXidKey, "")
	if rpcXid == "" {
		rpcXid = invocation.GetAttachmentWithDefaultValue(strings.ToLower(common.SeataXidKey), "")
	}
	return rpcXid
}

func (*DubboTransactionFilter) getDubboJavaRpcXid(invocation protocol.Invocation) string {
	rpcXid := invocation.GetAttachmentWithDefaultValue(common.XidKey, "")
	if rpcXid == "" {
		rpcXid = invocation.GetAttachmentWithDefaultValue(strings.ToLower(common.XidKey), "")
	}
	return rpcXid
}
