package rm

import (
	"github.com/seata/seata-go/pkg/common/model"
)

// TODO
type RMRemoting struct {
}

// Branch register long
func (RMRemoting) BranchRegister(branchType model.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return 0, nil
}

//  Branch report
func (RMRemoting) BranchReport(branchType model.BranchType, xid string, branchId int64, status model.BranchStatus, applicationData string) error {
	return nil
}

// Lock query boolean
func (RMRemoting) LockQuery(branchType model.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return false, nil
}

func (RMRemoting) RegisterResource(resource model.Resource) error {
	return nil
}
