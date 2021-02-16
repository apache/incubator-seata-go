package service

import (
	"fmt"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/context"
	"github.com/transaction-wg/seata-golang/pkg/client/tcc"
)

type ServiceA struct {
}

func (svc *ServiceA) Try(ctx *context.BusinessActionContext) (bool, error) {
	word := ctx.ActionContext["hello"]
	fmt.Println(word)
	fmt.Println("Service A Tried!")
	return true, nil
}

func (svc *ServiceA) Confirm(ctx *context.BusinessActionContext) bool {
	word := ctx.ActionContext["hello"]
	fmt.Println(word)
	fmt.Println("Service A confirmed!")
	return true
}

func (svc *ServiceA) Cancel(ctx *context.BusinessActionContext) bool {
	word := ctx.ActionContext["hello"]
	fmt.Println(word)
	fmt.Println("Service A canceled!")
	return true
}

var serviceA = &ServiceA{}

type TCCProxyServiceA struct {
	*ServiceA

	Try func(ctx *context.BusinessActionContext) (bool, error) `TccActionName:"ServiceA"`
}

func (svc *TCCProxyServiceA) GetTccService() tcc.TccService {
	return svc.ServiceA
}

var TccProxyServiceA = &TCCProxyServiceA{
	ServiceA: serviceA,
}
