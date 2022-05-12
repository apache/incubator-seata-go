package mock

import (
	"context"
	"fmt"
	"github.com/seata/seata-go/pkg/rm/tcc/remoting"
	xid_utils "github.com/seata/seata-go/pkg/utils/xid"
)

import (
	"github.com/seata/seata-go/pkg/rm/api"
	_ "github.com/seata/seata-go/pkg/utils/xid"
)

// 注册RM资源
func init() {

}

type MockTccService struct {
}

func (*MockTccService) Prepare(ctx context.Context, params interface{}) error {
	xid := xid_utils.GetXID(ctx)
	fmt.Printf("TccActionOne prepare, xid:" + xid)
	return nil
}

func (*MockTccService) Commit(ctx context.Context, businessActionContext api.BusinessActionContext) error {
	xid := xid_utils.GetXID(ctx)
	fmt.Printf("TccActionOne commit, xid:" + xid)
	return nil
}

func (*MockTccService) Rollback(ctx context.Context, businessActionContext api.BusinessActionContext) error {
	xid := xid_utils.GetXID(ctx)
	fmt.Printf("TccActionOne rollback, xid:" + xid)
	return nil
}

func (*MockTccService) GetRemoteType() remoting.RemoteType {
	return remoting.RemoteTypeLocalService
}

func (*MockTccService) GetActionName() string {
	return "MockTccService"
}

func (*MockTccService) GetServiceType() remoting.ServiceType {
	return remoting.ServiceTypeProvider
}
