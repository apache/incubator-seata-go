package test

import (
	"context"
	"github.com/seata/seata-go/pkg/common/model"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/rm/tcc/api"
	"github.com/seata/seata-go/pkg/rm/tcc/remoting"
	"github.com/seata/seata-go/pkg/utils/log"
	"testing"
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

func (T TestTCCServiceBusiness) GetRemoteType() remoting.RemoteType {
	return remoting.RemoteTypeLocalService
}

func (T TestTCCServiceBusiness) GetServiceType() remoting.ServiceType {
	return remoting.ServiceTypeProvider
}

func TestNew(test *testing.T) {
	tccService := tcc.NewTCCServiceProxy(TestTCCServiceBusiness{})
	tccService.Prepare(model.InitSeataContext(context.Background()), 1)

	//time.Sleep(time.Second * 1000)
}
