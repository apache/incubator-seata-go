package remoting

import (
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/resource"
	"github.com/seata/seata-go/pkg/remoting/getty"
	"sync"
)

var (
	rmRemoting        *RMRemoting
	onceGettyRemoting = &sync.Once{}
)

func GetRMRemotingInstance() *RMRemoting {
	if rmRemoting == nil {
		onceGettyRemoting.Do(func() {
			rmRemoting = &RMRemoting{}
		})
	}
	return rmRemoting
}

// TODO
type RMRemoting struct {
}

// Branch register long
func (RMRemoting) BranchRegister(branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return 0, nil
}

//  Branch report
func (RMRemoting) BranchReport(branchType branch.BranchType, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	return nil
}

// Lock query boolean
func (RMRemoting) LockQuery(branchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return false, nil
}

func (r *RMRemoting) RegisterResource(resource resource.Resource) error {
	req := message.RegisterRMRequest{
		AbstractIdentifyRequest: message.AbstractIdentifyRequest{
			//todo replace with config
			Version:                 "1.4.2",
			ApplicationId:           "tcc-sample",
			TransactionServiceGroup: "my_test_tx_group",
		},
		ResourceIds: resource.GetResourceId(),
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("RegisterResourceManager error: {%#v}", err.Error())
		return err
	}

	if isRegisterSuccess(res) {
		r.onRegisterRMSuccess(res.(message.RegisterRMResponse))
	} else {
		r.onRegisterRMFailure(res.(message.RegisterRMResponse))
	}

	return nil
}

func isRegisterSuccess(response interface{}) bool {
	//if res, ok := response.(protocol.RegisterTMResponse); ok {
	//	return res.Identified
	//} else if res, ok := response.(protocol.RegisterRMResponse); ok {
	//	return res.Identified
	//}
	//return false
	if res, ok := response.(message.RegisterRMResponse); ok {
		return res.Identified
	}
	return false
}

func (r *RMRemoting) onRegisterRMSuccess(response message.RegisterRMResponse) {
	// TODO
	log.Infof("register RM success. response: %#v", response)
}

func (r *RMRemoting) onRegisterRMFailure(response message.RegisterRMResponse) {
	// TODO
	log.Infof("register RM failure. response: %#v", response)
}

func (r *RMRemoting) onRegisterTMSuccess(response message.RegisterTMResponse) {
	// TODO
	log.Infof("register TM success. response: %#v", response)
}

func (r *RMRemoting) onRegisterTMFailure(response message.RegisterTMResponse) {
	// TODO
	log.Infof("register TM failure. response: %#v", response)
}
