package tcc

import (
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/rm"
)

type TCCRMHandler struct {
	rm.CommonRMHandler
}

func NewTCCRMHandler() TCCRMHandler {
	handler := TCCRMHandler{}
	handler.CommonRMHandler.SetRMGetter(handler)
	return handler
}

func (TCCRMHandler) GetResourceManager() model.ResourceManager {
	return rm.GetResourceManagerFacadeInstance().GetResourceManager(model.BranchTypeTCC)
}
