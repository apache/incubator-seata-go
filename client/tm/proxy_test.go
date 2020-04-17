package tm

import (
	"context"
	"github.com/dk-lockdown/seata-golang/client/config"
	getty2 "github.com/dk-lockdown/seata-golang/client/getty"
	"github.com/dk-lockdown/seata-golang/client/proxy"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Dog struct {
}

type Cat struct {

}

type ZooService struct {

}

func (svc *ZooService) ManageAnimal(ctx context.Context,dog *Dog,cat *Cat) (bool,error) {
	return true,nil
}

func (svc *ZooService) ProxyMethods() map[string]bool {
	mp := make(map[string]bool)
	mp["ManageAnimal"] = true
	return mp
}

type TestService struct {
	proxy.ProxyService
	ManageAnimal func(ctx context.Context,dog *Dog,cat *Cat) (bool,error)
}

var methodTransactionInfo = make(map[string]*TransactionInfo)

func init() {
	methodTransactionInfo["ManageAnimal"] = &TransactionInfo{
		TimeOut:     60000,
		Name:        "",
		Propagation: REQUIRED,
	}
}

func (svc *TestService) GetProxyService() proxy.ProxyService {
	return svc.ProxyService
}

func (svc *TestService) GetMethodTransactionInfo(methodName string) *TransactionInfo {
	return methodTransactionInfo[methodName]
}

func TestProxy_Implement(t *testing.T) {
	config.InitConfWithDefault("testService")
	getty2.InitRpcClient()
	NewTMClient()
	zooSvc := &ZooService{}
	proxy.ServiceMap.Register(zooSvc)

	ts := &TestService{ProxyService: zooSvc}
	Implement(ts)
	result, err := ts.ManageAnimal(context.Background(),&Dog{},&Cat{})
	assert.True(t,result)
	assert.Nil(t,err)
}