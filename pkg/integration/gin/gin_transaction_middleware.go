package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/tm"

	"github.com/seata/seata-go/pkg/common/log"
)

// TransactionMiddleware filter gin invocation
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

		retContext := tm.InitSeataContext(ctx)
		tm.SetXID(retContext, xid)
		log.Infof("global transaction xid is :%s", xid)

		ctx.Next()
	}
}
