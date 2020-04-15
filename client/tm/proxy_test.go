package proxy

import (
	"context"
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

func (svc *ZooService) TransactionMethods() map[string]bool {
	mp := make(map[string]bool)
	mp["ManageAnimal"] = true
	return mp
}

type TestService struct {
	GlobalTransactionService
	ManageAnimal func(ctx context.Context,dog *Dog,cat *Cat) (bool,error)
}

var methodTransactionInfo = make(map[string]TransactionInfo)

func init() {
	methodTransactionInfo["ManageAnimal"] = TransactionInfo{
		TimeOut:     60000,
		Name:        "",
		Propagation: REQUIRED,
	}
}

func (svc *TestService) GetProxiedService() GlobalTransactionService {
	return svc.GlobalTransactionService
}

func (svc *TestService) GetMethodTransactionInfo(methodName string) TransactionInfo {
	return methodTransactionInfo[methodName]
}

func TestProxy_Implement(t *testing.T) {
	zooSvc := &ZooService{}
	ServiceMap.Register(zooSvc)

	ts := &TestService{GlobalTransactionService:zooSvc}
	Implement(ts)
	result, err := ts.ManageAnimal(context.Background(),&Dog{},&Cat{})
	assert.True(t,result)
	assert.Nil(t,err)
}