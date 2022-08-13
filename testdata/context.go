package testdata

import (
	"context"

	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/tm"
)

const (
	TestTxName   = "TesttxName"
	TestXid      = "TestXid"
	TestXidCopy  = "TestXid"
	TestTxRole   = tm.LAUNCHER
	TestTxStatus = message.GlobalStatusBegin
)

func GetTestContext() context.Context {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, TestXid)
	tm.SetTxName(ctx, TestTxName)
	tm.SetXID(ctx, TestXid)
	tm.SetXIDCopy(ctx, TestXidCopy)
	tm.SetTransactionRole(ctx, TestTxRole)
	tm.SetTxStatus(ctx, TestTxStatus)
	return ctx
}
