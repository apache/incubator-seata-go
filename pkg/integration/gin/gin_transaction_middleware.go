package gin

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/tm"
)

// TransactionMiddleware filter gin invocation
// NOTE: when use gin，must set gin.ContextWithFallback true
func TransactionMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		xid := ctx.GetHeader(common.XidKey)
		if xid == "" {
			xid = ctx.GetHeader(common.XidKeyLowercase)
		}

		if len(xid) == 0 {
			log.Errorf("Gin: header not contain header: %s, global transaction xid", common.XidKey)
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		newCtx := context.Background()
		newCtx = tm.InitSeataContext(newCtx)
		tm.SetXID(newCtx, xid)
		ctx.Request = ctx.Request.WithContext(newCtx)

		log.Infof("global transaction xid is :%s", xid)
	}
}
