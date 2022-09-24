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
	flag.Parse()
	client.Init()
	bgCtx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	var serverIpPort = "http://127.0.0.1:8080"

	tm.WithGlobalTx(
		bgCtx,
		&tm.TransactionInfo{
			Name: "TccSampleLocalGlobalTx",
		},
		func(ctx context.Context) (re error) {
			request := gorequest.New()
			log.Infof("branch transaction begin")
			request.Post(serverIpPort+"/prepare").
				Set(common.XidKey, tm.GetXID(ctx)).
				End(func(response gorequest.Response, body string, errs []error) {
					if len(errs) != 0 {
						re = errs[0]
					}
				})
			return
		})
}
