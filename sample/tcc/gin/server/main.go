package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/common/log"
	ginmiddleware "github.com/seata/seata-go/pkg/integration/gin"
	"github.com/seata/seata-go/pkg/rm/tcc"
)

func main() {
	client.Init()

	r := gin.Default()
	r.Use(ginmiddleware.TransactionMiddleware())

	userProviderProxy, err := tcc.NewTCCServiceProxy(&RMService{})
	if err != nil {
		log.Errorf("get userProviderProxy tcc service proxy error, %v", err.Error())
		return
	}

	r.POST("/prepare", func(c *gin.Context) {
		if _, err := userProviderProxy.Prepare(c, nil); err != nil {
			c.JSON(http.StatusOK, "prepare failure")
			return
		}
		c.JSON(http.StatusOK, "prepare ok")
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("start tcc server fatal: %v", err)
	}
}
