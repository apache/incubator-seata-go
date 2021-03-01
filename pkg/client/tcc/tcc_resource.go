package tcc

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/client/proxy"
)

type TCCResource struct {
	ResourceGroupID    string
	AppName            string
	ActionName         string
	PrepareMethodName  string
	CommitMethodName   string
	CommitMethod       *proxy.MethodDescriptor
	RollbackMethodName string
	RollbackMethod     *proxy.MethodDescriptor
}

func (resource *TCCResource) GetResourceGroupID() string {
	return resource.ResourceGroupID
}

func (resource *TCCResource) GetResourceID() string {
	return resource.ActionName
}

func (resource *TCCResource) GetBranchType() meta.BranchType {
	return meta.BranchTypeTCC
}
