package saga

import (
	"context"
	gostnet "github.com/dubbogo/gost/net"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/util/log"
	"sync"
	"time"
)

type SagaServiceProxy struct {
	referenceName        string
	registerResourceOnce sync.Once
	*SagaResource
}

func NewSagaServiceProxy(service interface{}) (*SagaServiceProxy, error) {
	sagaResource, err := ParseSagaResource(service)
	if err != nil {
		log.Errorf("invalid saga service, err %v", err)
	}
	proxy := &SagaServiceProxy{
		SagaResource: sagaResource,
	}
	return proxy, proxy.RegisterResource()
}

func (t *SagaServiceProxy) RegisterResource() error {
	var err error
	t.registerResourceOnce.Do(func() {
		err = rm.GetRmCacheInstance().GetResourceManager(branch.BranchTypeSAGA).RegisterResource(t.SagaResource)
		if err != nil {
			log.Errorf("NewTCCServiceProxy RegisterResource error: %#v", err.Error())
		}
	})
	return err
}

func (proxy *SagaServiceProxy) initActionContext(params interface{}) map[string]interface{} {
	//只需要从上下文中携带action, compensationAction
	actionContext := proxy.getActionContextParameters(params)
	actionContext[constant.ActionStartTime] = time.Now().UnixNano() / 1e6
	actionContext[constant.ActionMethod] = proxy.SagaResource.SagaAction.GetNormalActionName()
	actionContext[constant.CompensationMethod] = proxy.SagaResource.SagaAction.GetCompensationName()
	actionContext[constant.ActionName] = proxy.SagaResource.SagaAction.GetActionName(params)
	actionContext[constant.HostName], _ = gostnet.GetLocalIP()
	return actionContext
}

func (proxy *SagaServiceProxy) registerBranch(ctx context.Context, params interface{}) error {
	return nil
}

func (proxy *SagaServiceProxy) getActionContextParameters(params interface{}) map[string]interface{} {
	return nil
}
