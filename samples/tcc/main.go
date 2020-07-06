package main

import (
	"github.com/dk-lockdown/seata-golang/samples/tcc/service"
	"github.com/gin-gonic/gin"
)

import (
	"github.com/dk-lockdown/seata-golang/client"
	"github.com/dk-lockdown/seata-golang/client/config"
	"github.com/dk-lockdown/seata-golang/client/tcc"
	"github.com/dk-lockdown/seata-golang/client/tm"
)

func main() {
	r := gin.Default()

	config.InitConfWithDefault("testService")
	client.NewRpcClient()
	tcc.InitTCCResourceManager()

	tm.Implement(service.ProxySvc)
	tcc.ImplementTCC(service.TccProxyServiceA)
	tcc.ImplementTCC(service.TccProxyServiceB)
	tcc.ImplementTCC(service.TccProxyServiceC)

	r.GET("/commit", func(c *gin.Context) {
		service.ProxySvc.TCCCommitted(c)
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/rollback", func(c *gin.Context) {
		service.ProxySvc.TCCCanceled(c)
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run()
}
