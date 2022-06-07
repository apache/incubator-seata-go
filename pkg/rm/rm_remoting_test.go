package rm

import (
	"github.com/seata/seata-go/pkg/common/model"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/rpc/getty"
	"github.com/seata/seata-go/pkg/utils/log"
)

func (RMRemoting) RegisterResource(resource model.Resource) error {
	req := protocol.RegisterRMRequest{
		AbstractIdentifyRequest: protocol.AbstractIdentifyRequest{
			//todo replace with config
			Version:                 "1.4.2",
			ApplicationId:           "tcc-sample",
			TransactionServiceGroup: "my_test_tx_group",
		},
		ResourceIds: resource.GetResourceId(),
	}
	err := getty.GetGettyRemotingClient().SendAsyncRequest(req)
	if err != nil {
		log.Error("RegisterResourceManager error: {%#v}", err.Error())
		return err
	}
	return nil
}
