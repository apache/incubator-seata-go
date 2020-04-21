package tm

import (
	"context"
	"github.com/dk-lockdown/seata-golang/client/config"
	getty2 "github.com/dk-lockdown/seata-golang/client/getty"
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

type TestService struct {
	*ZooService
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

func (svc *TestService) GetProxyService() interface{} {
	return svc.ZooService
}

func (svc *TestService) GetMethodTransactionInfo(methodName string) *TransactionInfo {
	return methodTransactionInfo[methodName]
}

func TestProxy_Implement(t *testing.T) {
	config.InitConfWithDefault("testService")
	getty2.InitRpcClient()
	NewTMClient()
	zooSvc := &ZooService{}

	ts := &TestService{ZooService: zooSvc}
	Implement(ts)
	result, err := ts.ManageAnimal(context.Background(),&Dog{},&Cat{})
	assert.True(t,result)
	assert.Nil(t,err)
}