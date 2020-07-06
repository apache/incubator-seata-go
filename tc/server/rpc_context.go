package server

import (
	"github.com/dubbogo/getty"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/base/model"
)

const IpPortSplitChar = ":"

type RpcContext struct {
	Version                 string
	TransactionServiceGroup string
	ClientRole              meta.TransactionRole
	ApplicationId           string
	ClientId                string
	ResourceSets            *model.Set
	Session                 getty.Session
}

type RpcContextOption func(ctx *RpcContext)

func WithRpcContextVersion(version string) RpcContextOption {
	return func(ctx *RpcContext) {
		ctx.Version = version
	}
}

func WithRpcContextTxServiceGroup(txServiceGroup string) RpcContextOption {
	return func(ctx *RpcContext) {
		ctx.TransactionServiceGroup = txServiceGroup
	}
}

func WithRpcContextClientRole(clientRole meta.TransactionRole) RpcContextOption {
	return func(ctx *RpcContext) {
		ctx.ClientRole = clientRole
	}
}

func WithRpcContextApplicationId(applicationId string) RpcContextOption {
	return func(ctx *RpcContext) {
		ctx.ApplicationId = applicationId
	}
}

func WithRpcContextClientId(clientId string) RpcContextOption {
	return func(ctx *RpcContext) {
		ctx.ClientId = clientId
	}
}

func WithRpcContextResourceSet(resourceSet *model.Set) RpcContextOption {
	return func(ctx *RpcContext) {
		ctx.ResourceSets = resourceSet
	}
}

func WithRpcContextSession(session getty.Session) RpcContextOption {
	return func(ctx *RpcContext) {
		ctx.Session = session
	}
}

func NewRpcContext(opts ...RpcContextOption) *RpcContext {
	ctx := &RpcContext{
		ResourceSets: model.NewSet(),
	}
	for _, o := range opts {
		o(ctx)
	}
	return ctx
}

func (context *RpcContext) AddResource(resource string) {
	if resource != "" {
		if context.ResourceSets == nil {
			context.ResourceSets = model.NewSet()
		}
		context.ResourceSets.Add(resource)
	}
}

func (context *RpcContext) AddResources(resources *model.Set) {
	if resources != nil {
		if context.ResourceSets == nil {
			context.ResourceSets = model.NewSet()
		}
		for _, resource := range resources.List() {
			context.ResourceSets.Add(resource)
		}
	}
}
