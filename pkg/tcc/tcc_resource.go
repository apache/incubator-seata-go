package tcc

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/proxy"
)

type TCCResource struct {
	ResourceGroupId    string
	AppName            string
	ActionName         string
	PrepareMethodName  string
	CommitMethodName   string
	CommitMethod       *proxy.MethodDescriptor
	RollbackMethodName string
	RollbackMethod     *proxy.MethodDescriptor
}

func (resource *TCCResource) GetResourceGroupId() string {
	return resource.ResourceGroupId
}

func (resource *TCCResource) GetResourceId() string {
	return resource.ActionName
}

func (resource *TCCResource) GetBranchType() meta.BranchType {
	return meta.BranchTypeTCC
}
