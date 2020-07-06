package tm

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dk-lockdown/seata-golang/client"
	"github.com/dk-lockdown/seata-golang/client/config"
)

type Dog struct {
}

type Cat struct {
}

type ZooService struct {
}

func (svc *ZooService) ManageAnimal(ctx context.Context, dog *Dog, cat *Cat) (bool, error) {
	return true, nil
}

type TestService struct {
	*ZooService
	ManageAnimal func(ctx context.Context, dog *Dog, cat *Cat) (bool, error)
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
	client.NewRpcClient()
	zooSvc := &ZooService{}

	ts := &TestService{ZooService: zooSvc}
	Implement(ts)
	result, err := ts.ManageAnimal(context.Background(), &Dog{}, &Cat{})
	assert.True(t, result)
	assert.Nil(t, err)
}
