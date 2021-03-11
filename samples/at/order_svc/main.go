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
	"github.com/transaction-wg/seata-golang/pkg/client"
	"github.com/transaction-wg/seata-golang/pkg/client/at/exec"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	"github.com/transaction-wg/seata-golang/pkg/client/context"
	"github.com/transaction-wg/seata-golang/samples/at/order_svc/dao"
)

const configPath = "/Users/scottlewis/dksl/git/1/seata-golang/samples/at/order_svc/conf/client.yml"

func main() {
	r := gin.Default()
	config.InitConf(configPath)
	client.NewRpcClient()
	exec.InitDataResourceManager()

	sqlDB, err := sql.Open("mysql", config.GetATConfig().DSN)
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(4 * time.Hour)

	db, err := exec.NewDB(config.GetATConfig(), sqlDB)
	if err != nil {
		panic(err)
	}
	d := &dao.Dao{
		DB: db,
	}

	r.POST("/createSo", func(c *gin.Context) {
		type req struct {
			Req []*dao.SoMaster
		}
		var q req
		if err := c.ShouldBindJSON(&q); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		rootContext := &context.RootContext{Context: c}
		rootContext.Bind(c.Request.Header.Get("XID"))

		_, err := d.CreateSO(rootContext, q.Req)

		if err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "fail",
			})
		} else {
			c.JSON(200, gin.H{
				"success": true,
				"message": "success",
			})
		}
	})
	r.Run(":8002")
}
