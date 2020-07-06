package main

import (
	"database/sql"
	"net/http"
	"time"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	"github.com/dk-lockdown/seata-golang/client"
	"github.com/dk-lockdown/seata-golang/client/at/exec"
	"github.com/dk-lockdown/seata-golang/client/config"
	"github.com/dk-lockdown/seata-golang/client/context"
	"github.com/dk-lockdown/seata-golang/samples/at/product_svc/dao"
)

const configPath = "/Users/scottlewis/dksl/git/1/seata-golang/samples/at/product_svc/conf/client.yml"

func main() {
	r := gin.Default()
	config.InitConf(configPath)
	client.NewRpcClient()
	exec.InitDataResourceManager()

	sqlDB, err := sql.Open("mysql", config.GetATConfig().DSN)
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(4 * time.Hour)

	db, err := exec.NewDB(config.GetATConfig(), sqlDB)
	if err != nil {
		panic(err)
	}
	d := &dao.Dao{
		DB: db,
	}

	r.POST("/allocateInventory", func(c *gin.Context) {
		type req struct {
			Req []*dao.AllocateInventoryReq
		}
		var q req
		if err := c.ShouldBindJSON(&q); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rootContext := &context.RootContext{Context: c}
		rootContext.Bind(c.Request.Header.Get("Xid"))

		d.AllocateInventory(rootContext, q.Req)

		c.JSON(200, gin.H{
			"success": true,
			"message": "success",
		})
	})

	r.Run(":8001")
}
