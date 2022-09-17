package main

import (
	"context"
	"flag"
	"time"

	"github.com/parnurzeal/gorequest"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/tm"
)

func main() {
	client.Init()
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var err error

	// GlobalTransactional is starting
	log.Infof("global transaction begin")
	ctx = tm.Begin(ctx, "TestTCCServiceBusiness")
	defer func() {
		resp := tm.CommitOrRollback(ctx, err == nil)
		log.Infof("tx result %v", resp)
		<-make(chan bool)
	}()

	var serverIpPort = "http://127.0.0.1:8080"
	request := gorequest.New()

	var xid string
	if tm.IsSeataContext(ctx) {
		xid = tm.GetXID(ctx)
	}

	log.Infof("branch transaction begin")
	request.Post(serverIpPort+"/prepare").
		Set(common.XidKey, xid).
		End(func(response gorequest.Response, body string, errs []error) {
			if len(errs) != 0 {
				err = errs[0]
			}
		})
}
