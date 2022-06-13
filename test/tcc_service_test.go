package test

import (
	"context"
	"testing"
	"time"
)

import (
	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	txapi "github.com/seata/seata-go/pkg/protocol/transaction/api"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/rm/tcc/api"
)

type TestTCCServiceBusiness struct {
}

func (T TestTCCServiceBusiness) Prepare(ctx context.Context, params interface{}) error {
	log.Infof("TestTCCServiceBusiness Prepare, param %v", params)
	return nil
}

func (T TestTCCServiceBusiness) Commit(ctx context.Context, businessActionContext api.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Commit, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness) Rollback(ctx context.Context, businessActionContext api.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Rollback, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness) GetActionName() string {
	return "TestTCCServiceBusiness"
}

type TestTCCServiceBusiness2 struct {
}

func (T TestTCCServiceBusiness2) Prepare(ctx context.Context, params interface{}) error {
	log.Infof("TestTCCServiceBusiness2 Prepare, param %v", params)
	return nil
}

func (T TestTCCServiceBusiness2) Commit(ctx context.Context, businessActionContext api.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness2 Commit, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness2) Rollback(ctx context.Context, businessActionContext api.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness2 Rollback, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness2) GetActionName() string {
	return "TestTCCServiceBusiness2"
}

func TestNew(test *testing.T) {
	var err error
	ctx := txapi.Begin(context.Background(), "TestTCCServiceBusiness")
	defer func() {
		resp := txapi.CommitOrRollback(ctx, err)
		log.Infof("tx result %v", resp)
	}()

	tccService := tcc.NewTCCServiceProxy(TestTCCServiceBusiness{})
	err = tccService.Prepare(ctx, 1)

	tccService2 := tcc.NewTCCServiceProxy(TestTCCServiceBusiness2{})
	err = tccService2.Prepare(ctx, 3)

	time.Sleep(time.Second * 1000)
}
