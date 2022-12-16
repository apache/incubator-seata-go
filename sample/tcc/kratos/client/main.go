package main

import (
	"context"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

func main() {
	client.Init()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := &tm.GtxConfig{Name: "TccSample Kratos GlobalTx"}

	tm.WithGlobalTx(ctx, cfg, transferAccounts)
}

func transferAccounts(ctx context.Context) (re error) {
	request := gorequest.New()
	serverIpPort := "http://127.0.0.1:8000"

	log.Infof("transfer from account A")
	request.Post(serverIpPort+"/transFrom").
		Set(constant.XidKey, tm.GetXID(ctx)).
		End(func(response gorequest.Response, body string, errs []error) {
			if len(errs) != 0 {
				re = errs[0]
			}
		})

	if re != nil {
		log.Infof("Failed to transfer from account A: %v", re)
		return
	}

	log.Infof("transfer to account B")
	request.Post(serverIpPort+"/transTo").
		Set(constant.XidKey, tm.GetXID(ctx)).
		End(func(response gorequest.Response, body string, errs []error) {
			if len(errs) != 0 {
				re = errs[0]
			}
		})

	if re != nil {
		log.Infof("Failed to transfer to account B: %v", re)
		return
	}

	return
}
