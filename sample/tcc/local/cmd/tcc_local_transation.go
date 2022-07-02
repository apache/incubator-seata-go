package main

import (
	"context"

	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/local/service"
)

func main() {

	var err error
	ctx := tm.Begin(context.Background(), "TestTCCServiceBusiness")
	defer func() {
		resp := tm.CommitOrRollback(ctx, err)
		log.Infof("tx result %v", resp)
		<-make(chan struct{})
	}()

	tccService := tcc.NewTCCServiceProxy(service.TestTCCServiceBusiness{})
	err = tccService.Prepare(ctx, 1)
	if err != nil {
		log.Errorf("execute TestTCCServiceBusiness prepare error %s", err.Error())
		return
	}

	tccService2 := tcc.NewTCCServiceProxy(service.TestTCCServiceBusiness2{})
	err = tccService2.Prepare(ctx, 3)
	if err != nil {
		log.Errorf("execute TestTCCServiceBusiness2 prepare error %s", err.Error())
		return
	}

}
