package service

import (
	"errors"
	"fmt"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/context"
	"github.com/transaction-wg/seata-golang/pkg/client/tcc"
)

type ServiceC struct {
}

func (svc *ServiceC) Try(ctx *context.BusinessActionContext) (bool, error) {
	word := ctx.ActionContext["hello"]
	fmt.Println(word)
	fmt.Println("Service C Tried!")
	return true, errors.New("there is a error")
}

func (svc *ServiceC) Confirm(ctx *context.BusinessActionContext) bool {
	word := ctx.ActionContext["hello"]
	fmt.Println(word)
	fmt.Println("Service C confirmed!")
	return true
}

func (svc *ServiceC) Cancel(ctx *context.BusinessActionContext) bool {
	word := ctx.ActionContext["hello"]
	fmt.Println(word)
	fmt.Println("Service C canceled!")
	return true
}

var serviceC = &ServiceC{}

type TCCProxyServiceC struct {
	*ServiceC

	Try func(ctx *context.BusinessActionContext) (bool, error) `TccActionName:"ServiceC"`
}

func (svc *TCCProxyServiceC) GetTccService() tcc.TccService {
	return svc.ServiceC
}

var TccProxyServiceC = &TCCProxyServiceC{
	ServiceC: serviceC,
}
