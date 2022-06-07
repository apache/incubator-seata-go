package handler

import (
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/resource"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/rm/common/handler"
)

type TCCRMHandler struct {
	handler.CommonRMHandler
}

func NewTCCRMHandler() TCCRMHandler {
	handler := TCCRMHandler{}
	handler.CommonRMHandler.SetRMGetter(handler)
	return handler
}

func (TCCRMHandler) GetResourceManager() resource.ResourceManager {
	return rm.GetResourceManagerInstance().GetResourceManager(branch.BranchTypeTCC)
}
