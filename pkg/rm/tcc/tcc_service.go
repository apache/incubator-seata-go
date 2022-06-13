package tcc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/common/net"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/seatactx"
	context2 "github.com/seata/seata-go/pkg/protocol/transaction"
	"github.com/seata/seata-go/pkg/rm"
	api2 "github.com/seata/seata-go/pkg/rm/tcc/api"
)

type TCCService interface {
	Prepare(ctx context.Context, params interface{}) error
	Commit(ctx context.Context, businessActionContext api2.BusinessActionContext) error
	Rollback(ctx context.Context, businessActionContext api2.BusinessActionContext) error

	GetActionName() string
	//GetRemoteType() remoting.RemoteType
	//GetServiceType() remoting.ServiceType
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
	err := rm.GetResourceManagerInstance().GetResourceManager(branch.BranchTypeTCC).RegisterResource(&tccResource)
	if err != nil {
		panic(fmt.Sprintf("NewTCCServiceProxy registerResource error: {%#v}", err.Error()))
	}

	return &TCCServiceProxy{
		TCCService: tccService,
	}
}

func (t *TCCServiceProxy) Prepare(ctx context.Context, param interface{}) error {
	if seatactx.HasXID(ctx) {
		err := t.RegisteBranch(ctx, param)
		if err != nil {
			return err
		}
	}
	return t.TCCService.Prepare(ctx, param)
}

func (t *TCCServiceProxy) RegisteBranch(ctx context.Context, param interface{}) error {
	// register transaction branch
	if !seatactx.HasXID(ctx) {
		err := errors.New("BranchRegister error, xid should not be nil")
		log.Errorf(err.Error())
		return err
	}
	tccContext := make(map[string]interface{}, 0)
	tccContext[common.StartTime] = time.Now().UnixNano() / 1e6
	tccContext[common.HostName] = net.GetLocalIp()
	tccContextStr, _ := json.Marshal(tccContext)

	branchId, err := rm.GetResourceManagerInstance().GetResourceManager(branch.BranchTypeTCC).BranchRegister(
		ctx, branch.BranchTypeTCC, t.GetActionName(), "", seatactx.GetXID(ctx), string(tccContextStr), "")
	if err != nil {
		err = errors.New(fmt.Sprintf("BranchRegister error: %v", err.Error()))
		log.Error(err.Error())
		return err
	}

	actionContext := &api2.BusinessActionContext{
		Xid:           seatactx.GetXID(ctx),
		BranchId:      string(branchId),
		ActionName:    t.GetActionName(),
		ActionContext: param,
	}
	seatactx.SetBusinessActionContext(ctx, actionContext)
	return nil
}

func (t *TCCServiceProxy) GetTransactionInfo() context2.TransactionInfo {
	// todo replace with config
	return context2.TransactionInfo{
		TimeOut: 10000,
		Name:    t.GetActionName(),
		//Propagation, Propagation
		//LockRetryInternal, int64
		//LockRetryTimes    int64
	}
}
