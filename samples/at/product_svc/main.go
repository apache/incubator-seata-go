package main

import (
	"net/http"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	"github.com/xiaobudongzhang/seata-golang/client"
	"github.com/xiaobudongzhang/seata-golang/client/at/exec"
	"github.com/xiaobudongzhang/seata-golang/client/at/sql/struct/cache"
	"github.com/xiaobudongzhang/seata-golang/client/config"
	"github.com/xiaobudongzhang/seata-golang/client/context"
	"github.com/xiaobudongzhang/seata-golang/samples/at/product_svc/dao"
)

const configPath="/Users/scottlewis/dksl/git/1/seata-golang/samples/at/product_svc/conf/client.yml"

func main() {
	r := gin.Default()
    config.InitConf(configPath)
	client.NewRpcClient()
	cache.SetTableMetaCache(cache.NewMysqlTableMetaCache(config.GetClientConfig().ATConfig.DSN))
	exec.InitDataResourceManager()

	db,err := exec.NewDB(config.GetClientConfig().ATConfig)
	if err != nil {
		panic(err)
	}
	d := &dao.Dao{
		DB: db,
	}

	r.POST("/allocateInventory", func(c *gin.Context) {
		type req struct{
			Req []*dao.AllocateInventoryReq
		}
		var q req
		if err := c.ShouldBindJSON(&q); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rootContext := &context.RootContext{Context:c}
		rootContext.Bind(c.Request.Header.Get("Xid"))

		d.AllocateInventory(rootContext,q.Req)

		c.JSON(200, gin.H{
			"success": true,
			"message": "success",
		})
	})

	r.Run(":8001")
}
