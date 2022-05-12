package xid_utils

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/common"
)

func GetXID(ctx context.Context) string {
	xid := ctx.Value(common.XID)
	if xid == nil {
		return ""
	}
	return xid.(string)
}

func HasXID(ctx context.Context) bool {
	return GetXID(ctx) != ""
}

func SetXID(ctx context.Context, xid string) context.Context {
	return context.WithValue(ctx, common.XID, xid)
}
