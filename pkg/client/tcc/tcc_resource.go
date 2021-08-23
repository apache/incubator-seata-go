package tcc

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/client/proxy"
)

type Resource struct {
	ActionName         string
	PrepareMethodName  string
	CommitMethodName   string
	CommitMethod       *proxy.MethodDescriptor
	RollbackMethodName string
	RollbackMethod     *proxy.MethodDescriptor
}

func (resource *Resource) GetResourceID() string {
	return resource.ActionName
}

func (resource *Resource) GetBranchType() apis.BranchSession_BranchType {
	return apis.TCC
}
