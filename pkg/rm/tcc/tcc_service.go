package tcc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/rm"
	api2 "github.com/seata/seata-go/pkg/rm/tcc/api"
	"github.com/seata/seata-go/pkg/utils"
	"github.com/seata/seata-go/pkg/utils/log"
	"time"
)

import (
	"github.com/seata/seata-go/pkg/rm/api"
	"github.com/seata/seata-go/pkg/rm/tcc/remoting"
)

type TCCService interface {
	Prepare(ctx context.Context, params interface{}) error
	Commit(ctx context.Context, businessActionContext api2.BusinessActionContext) error
	Rollback(ctx context.Context, businessActionContext api2.BusinessActionContext) error

	GetActionName() string
	GetRemoteType() remoting.RemoteType
	GetServiceType() remoting.ServiceType
}

type TCCServiceProxy struct {
	TCCService
}

func NewTCCServiceProxy(tccService TCCService) TCCService {
	if tccService == nil {
		panic("param tccService should not be nil")
	}

	// register resource
	tccResource := TCCResource{
		TCCServiceBean:  tccService,
		ResourceGroupId: "DEFAULT",
		AppName:         "",
		ActionName:      tccService.GetActionName(),
	}
	err := rm.GetResourceManagerFacadeInstance().GetResourceManager(model.BranchTypeTCC).RegisterResource(&tccResource)
	if err != nil {
		panic(fmt.Sprintf("NewTCCServiceProxy registerResource error: {%#v}", err.Error()))
	}

	return &TCCServiceProxy{
		TCCService: tccService,
	}
}

func (t *TCCServiceProxy) Prepare(ctx context.Context, param interface{}) error {
	var err error
	if model.IsSeataContext(ctx) {
		// execute transaction
		_, err = api.GetTransactionTemplate().Execute(ctx, t, param)
	} else {
		log.Warn("context is not inited as seata context, will not execute transaction!")
		err = t.TCCService.Prepare(ctx, param)
	}
	return err
}

// register transaction branch, and then execute business
func (t *TCCServiceProxy) Execute(ctx context.Context, param interface{}) (interface{}, error) {
	// register transaction branch
	err := t.RegisteBranch(ctx, param)
	if err != nil {
		return nil, err
	}
	return nil, t.TCCService.Prepare(ctx, param)
}

func (t *TCCServiceProxy) RegisteBranch(ctx context.Context, param interface{}) error {
	// register transaction branch
	if !model.HasXID(ctx) {
		err := errors.New("BranchRegister error, xid should not be nil")
		log.Errorf(err.Error())
		return err
	}
	tccContext := make(map[string]interface{}, 0)
	tccContext[common.StartTime] = time.Now().UnixNano() / 1e6
	tccContext[common.HostName] = utils.GetLocalIp()
	tccContextStr, _ := json.Marshal(tccContext)

	branchId, err := rm.GetResourceManagerFacadeInstance().GetResourceManager(model.BranchTypeTCC).BranchRegister(
		ctx, model.BranchTypeTCC, t.GetActionName(), "", model.GetXID(ctx), string(tccContextStr), "")
	if err != nil {
		err = errors.New(fmt.Sprintf("BranchRegister error: %v", err.Error()))
		log.Error(err.Error())
		return err
	}

	actionContext := &api2.BusinessActionContext{
		Xid:           model.GetXID(ctx),
		BranchId:      string(branchId),
		ActionName:    t.GetActionName(),
		ActionContext: param,
	}
	model.SetBusinessActionContext(ctx, actionContext)
	return nil
}

func (t *TCCServiceProxy) GetTransactionInfo() api.TransactionInfo {
	// todo replace with config
	return api.TransactionInfo{
		TimeOut: 10000,
		Name:    t.GetActionName(),
		//Propagation, Propagation
		//LockRetryInternal, int64
		//LockRetryTimes    int64
	}
}
