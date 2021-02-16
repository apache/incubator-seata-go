package tm

import (
	"fmt"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	context2 "github.com/transaction-wg/seata-golang/pkg/client/context"
	"github.com/transaction-wg/seata-golang/pkg/client/rpc_client"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

const (
	DEFAULT_GLOBAL_TX_TIMEOUT = 60000
	DEFAULT_GLOBAL_TX_NAME    = "default"
)

type SuspendedResourcesHolder struct {
	Xid string
}

type GlobalTransaction interface {
	Begin(ctx *context2.RootContext) error
	BeginWithTimeout(timeout int32, ctx *context2.RootContext) error
	BeginWithTimeoutAndName(timeout int32, name string, ctx *context2.RootContext) error
	Commit(ctx *context2.RootContext) error
	Rollback(ctx *context2.RootContext) error
	Suspend(unbindXid bool, ctx *context2.RootContext) (*SuspendedResourcesHolder, error)
	Resume(suspendedResourcesHolder *SuspendedResourcesHolder, ctx *context2.RootContext) error
	GetStatus(ctx *context2.RootContext) (meta.GlobalStatus, error)
	GetXid(ctx *context2.RootContext) string
	GlobalReport(globalStatus meta.GlobalStatus, ctx *context2.RootContext) error
	GetLocalStatus() meta.GlobalStatus
}

type GlobalTransactionRole byte

const (
	// The Launcher. The one begins the current global transaction.
	Launcher GlobalTransactionRole = iota

	// The Participant. The one just joins into a existing global transaction.
	Participant
)

func (role GlobalTransactionRole) String() string {
	switch role {
	case Launcher:
		return "Launcher"
	case Participant:
		return "Participant"
	default:
		return fmt.Sprintf("%d", role)
	}
}

type DefaultGlobalTransaction struct {
	conf               config.TMConfig
	Xid                string
	Status             meta.GlobalStatus
	Role               GlobalTransactionRole
	transactionManager TransactionManager
}

func (gtx *DefaultGlobalTransaction) Begin(ctx *context2.RootContext) error {
	return gtx.BeginWithTimeout(DEFAULT_GLOBAL_TX_TIMEOUT, ctx)
}

func (gtx *DefaultGlobalTransaction) BeginWithTimeout(timeout int32, ctx *context2.RootContext) error {
	return gtx.BeginWithTimeoutAndName(timeout, DEFAULT_GLOBAL_TX_NAME, ctx)
}

func (gtx *DefaultGlobalTransaction) BeginWithTimeoutAndName(timeout int32, name string, ctx *context2.RootContext) error {
	if gtx.Role != Launcher {
		if gtx.Xid == "" {
			return errors.New("xid should not be empty")
		}
		log.Debugf("Ignore Begin(): just involved in global transaction [%s]", gtx.Xid)
		return nil
	}
	if gtx.Xid != "" {
		return errors.New("xid should be empty")
	}
	if ctx.InGlobalTransaction() {
		return errors.New("xid should be empty")
	}
	xid, err := gtx.transactionManager.Begin("", "", name, timeout)
	if err != nil {
		return errors.WithStack(err)
	}
	gtx.Xid = xid
	gtx.Status = meta.GlobalStatusBegin
	ctx.Bind(xid)
	log.Infof("Begin new global transaction [%s]", xid)
	return nil
}

func (gtx *DefaultGlobalTransaction) Commit(ctx *context2.RootContext) error {
	defer func() {
		ctxXid := ctx.GetXID()
		if ctxXid != "" && gtx.Xid == ctxXid {
			ctx.Unbind()
		}
	}()
	if gtx.Role == Participant {
		log.Debugf("Ignore Commit(): just involved in global transaction [%s]", gtx.Xid)
		return nil
	}
	if gtx.Xid == "" {
		return errors.New("xid should not be empty")
	}
	retry := gtx.conf.CommitRetryCount
	for retry > 0 {
		status, err := gtx.transactionManager.Commit(gtx.Xid)
		if err != nil {
			log.Errorf("Failed to report global commit [%s],Retry Countdown: %d, reason: %s", gtx.Xid, retry, err.Error())
		} else {
			gtx.Status = status
			break
		}
		retry--
		if retry == 0 {
			return errors.New("Failed to report global commit")
		}
	}
	log.Infof("[%s] commit status: %s", gtx.Xid, gtx.Status.String())
	return nil
}

func (gtx *DefaultGlobalTransaction) Rollback(ctx *context2.RootContext) error {
	defer func() {
		ctxXid := ctx.GetXID()
		if ctxXid != "" && gtx.Xid == ctxXid {
			ctx.Unbind()
		}
	}()
	if gtx.Role == Participant {
		log.Debugf("Ignore Rollback(): just involved in global transaction [%s]", gtx.Xid)
		return nil
	}
	if gtx.Xid == "" {
		return errors.New("xid should not be empty")
	}
	retry := gtx.conf.RollbackRetryCount
	for retry > 0 {
		status, err := gtx.transactionManager.Rollback(gtx.Xid)
		if err != nil {
			log.Errorf("Failed to report global rollback [%s],Retry Countdown: %d, reason: %s", gtx.Xid, retry, err.Error())
		} else {
			gtx.Status = status
			break
		}
		retry--
		if retry == 0 {
			return errors.New("Failed to report global rollback")
		}
	}
	log.Infof("[%s] rollback status: %s", gtx.Xid, gtx.Status.String())
	return nil
}

func (gtx *DefaultGlobalTransaction) Suspend(unbindXid bool, ctx *context2.RootContext) (*SuspendedResourcesHolder, error) {
	xid := ctx.GetXID()
	if xid != "" && unbindXid {
		ctx.Unbind()
		log.Debugf("Suspending current transaction,xid = %s", xid)
	} else {
		xid = ""
	}
	return &SuspendedResourcesHolder{Xid: xid}, nil
}

func (gtx *DefaultGlobalTransaction) Resume(suspendedResourcesHolder *SuspendedResourcesHolder, ctx *context2.RootContext) error {
	if suspendedResourcesHolder == nil {
		return nil
	}
	xid := suspendedResourcesHolder.Xid
	if xid != "" {
		ctx.Bind(xid)
		log.Debugf("Resumimg the transaction,xid = %s", xid)
	}
	return nil
}

func (gtx *DefaultGlobalTransaction) GetStatus(ctx *context2.RootContext) (meta.GlobalStatus, error) {
	if gtx.Xid == "" {
		return meta.GlobalStatusUnknown, nil
	}
	status, err := gtx.transactionManager.GetStatus(gtx.Xid)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	gtx.Status = status
	return gtx.Status, nil
}

func (gtx *DefaultGlobalTransaction) GetXid(ctx *context2.RootContext) string {
	return gtx.Xid
}

func (gtx *DefaultGlobalTransaction) GlobalReport(globalStatus meta.GlobalStatus, ctx *context2.RootContext) error {
	defer func() {
		ctxXid := ctx.GetXID()
		if ctxXid != "" && gtx.Xid == ctxXid {
			ctx.Unbind()
		}
	}()

	if gtx.Xid == "" {
		return errors.New("xid should not be empty")
	}

	if globalStatus == 0 {
		return errors.New("globalStatus should not be zero")
	}

	status, err := gtx.transactionManager.GlobalReport(gtx.Xid, globalStatus)
	if err != nil {
		return errors.WithStack(err)
	}

	gtx.Status = status
	log.Infof("[%s] report status: %s", gtx.Xid, gtx.Status.String())
	return nil
}

func (gtx *DefaultGlobalTransaction) GetLocalStatus() meta.GlobalStatus {
	return gtx.Status
}

func CreateNew() *DefaultGlobalTransaction {
	return &DefaultGlobalTransaction{
		conf:               config.GetTMConfig(),
		Xid:                "",
		Status:             meta.GlobalStatusUnknown,
		Role:               Launcher,
		transactionManager: &DefaultTransactionManager{rpcClient: rpc_client.GetRpcRemoteClient()},
	}
}

func GetCurrent(ctx *context2.RootContext) *DefaultGlobalTransaction {
	xid := ctx.GetXID()
	if xid == "" {
		return nil
	}
	return &DefaultGlobalTransaction{
		conf:               config.GetTMConfig(),
		Xid:                xid,
		Status:             meta.GlobalStatusBegin,
		Role:               Participant,
		transactionManager: &DefaultTransactionManager{rpcClient: rpc_client.GetRpcRemoteClient()},
	}
}

func GetCurrentOrCreate(ctx *context2.RootContext) *DefaultGlobalTransaction {
	tx := GetCurrent(ctx)
	if tx == nil {
		return CreateNew()
	}
	return tx
}
