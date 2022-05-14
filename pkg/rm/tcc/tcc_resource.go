package tcc

import (
	"reflect"
)

import (
	"github.com/seata/seata-go/pkg/common/model"
)

type TCCResource struct {
	ResourceGroupId    string `default:"DEFAULT"`
	AppName            string
	ActionName         string
	TargetBean         interface{}
	PrepareMethod      reflect.Method
	CommitMethodName   string
	CommitMethod       reflect.Method
	RollbackMethodName string
	RollbackMethod     reflect.Method
}

func (t *TCCResource) GetResourceGroupId() string {
	return t.ResourceGroupId
}

func (t *TCCResource) GetResourceId() string {
	return t.ActionName
}

func (t *TCCResource) GetBranchType() model.BranchType {
	return model.BranchTypeTCC
}
