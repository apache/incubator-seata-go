package service

import (
	"fmt"
)

import (
	"github.com/dk-lockdown/seata-golang/client/context"
	"github.com/dk-lockdown/seata-golang/client/tcc"
)

type ServiceC struct {

}

func (svc *ServiceC) Try(ctx *context.BusinessActionContext) (bool,error) {
	word := ctx.ActionContext["hello"]
	fmt.Print(word)
	fmt.Print("Service C Tried!")
	return true,nil
}

func (svc *ServiceC) Confirm(ctx *context.BusinessActionContext) bool {
	word := ctx.ActionContext["hello"]
	fmt.Print(word)
	fmt.Print("Service C confirmed!")
	return true
}

func (svc *ServiceC) Cancel(ctx *context.BusinessActionContext) bool {
	word := ctx.ActionContext["hello"]
	fmt.Print(word)
	fmt.Print("Service C canceled!")
	return true
}

var serviceC = &ServiceC{}

type TCCProxyServiceC struct {
	*ServiceC

	Try func(ctx *context.BusinessActionContext) (bool,error) `TccActionName:"ServiceC"`
}

func (svc *TCCProxyServiceC) GetTccService() tcc.TccService {
	return svc.ServiceC
}

var TccProxyServiceC = &TCCProxyServiceC{
	ServiceC: serviceC,
}