package remoting

import (
	"github.com/seata/seata-go/pkg/common/log"
	_ "github.com/seata/seata-go/pkg/imports"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/resource"
	"github.com/seata/seata-go/pkg/remoting/getty"
)

func (RMRemoting) RegisterResource(resource resource.Resource) error {
	req := message.RegisterRMRequest{
		AbstractIdentifyRequest: message.AbstractIdentifyRequest{
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
