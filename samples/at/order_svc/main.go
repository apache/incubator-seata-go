package main

import (
	"net/http"
	"time"

	"github.com/transaction-wg/seata-golang/pkg"
	"github.com/transaction-wg/seata-golang/pkg/at/exec"
	"github.com/transaction-wg/seata-golang/pkg/config"
	"github.com/transaction-wg/seata-golang/pkg/context"
	"github.com/transaction-wg/seata-golang/samples/at/order_svc/dao"

	"github.com/gin-gonic/gin"
	"xorm.io/xorm"
)

const configPath = "./conf/client.yml"

func main() {
	r := gin.Default()
	config.InitConf(configPath)
	pkg.NewRpcClient()
	exec.InitDataResourceManager()

	eng, err := xorm.NewEngine("mysql", config.GetATConfig().DSN)
	if err != nil {
		panic(err)
	}
	eng.SetMaxOpenConns(10)
	eng.SetMaxIdleConns(10)
	eng.SetConnMaxLifetime(4 * time.Hour)
	mydb, err := exec.NewDBWithXORM(config.GetATConfig(), eng)
	if err != nil {
		panic(err)
	}
	d := &dao.Dao{
		DB: mydb,
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
		rootContext.Bind(c.Request.Header.Get("Xid"))

		d.CreateSO(rootContext, q.Req)

		c.JSON(200, gin.H{
			"success": true,
			"message": "success",
		})
	})
	r.Run(":8002")
}
