package tm

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/rpc/getty"
	"github.com/seata/seata-go/pkg/tm/api"
	"github.com/seata/seata-go/pkg/utils/log"
	"sync"
)

type GlobalTransaction struct {
	Xid    string
	Status model.GlobalStatus
	Role   model.GlobalTransactionRole
}

var (
	// singletone ResourceManagerFacade
	globalTransactionManager     *GlobalTransactionManager
	onceGlobalTransactionManager = &sync.Once{}
)

func GetGlobalTransactionManager() *GlobalTransactionManager {
	if globalTransactionManager == nil {
		onceGlobalTransactionManager.Do(func() {
			globalTransactionManager = &GlobalTransactionManager{}
		})
	}
	return globalTransactionManager
}

type GlobalTransactionManager struct {
}

// Begin a new global transaction with given timeout and given name.
func (g *GlobalTransactionManager) Begin(ctx context.Context, transaction *GlobalTransaction, timeout int32, name string) error {
	if transaction.Role != model.LAUNCHER {
		log.Infof("Ignore Begin(): just involved in global transaction %s", transaction.Xid)
		return nil
	}
	if transaction.Xid != "" {
		return errors.New(fmt.Sprintf("Global transaction already exists,can't begin a new global transaction, currentXid = %s ", transaction.Xid))
	}

	req := protocol.GlobalBeginRequest{
		TransactionName: name,
		Timeout:         timeout,
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("GlobalBeginRequest error, xid %s, error %v", transaction.Xid, err)
		return err
	}
	if res == nil || res.(protocol.GlobalBeginResponse).ResultCode == protocol.ResultCodeFailed {
		log.Errorf("GlobalBeginRequest error, xid %s, res %v", transaction.Xid, res)
		return err
	}
	log.Infof("GlobalBeginRequest success, xid %s, res %v", transaction.Xid, res)

	transaction.Status = model.Begin
	transaction.Xid = res.(protocol.GlobalBeginResponse).Xid
	model.SetXID(ctx, res.(protocol.GlobalBeginResponse).Xid)
	return nil
}

// Commit the global transaction.
func (g *GlobalTransactionManager) Commit(ctx context.Context, transaction *GlobalTransaction) error {
	if transaction.Role != model.LAUNCHER {
		log.Infof("Ignore Commit(): just involved in global transaction [{}]", transaction.Xid)
		return nil
	}
	if transaction.Xid == "" {
		return errors.New("Commit xid should not be empty")
	}

	// todo: replace retry with config
	var (
		err error
		res interface{}
	)
	for retry := 5; retry > 0; retry-- {
		req := protocol.GlobalCommitRequest{
			AbstractGlobalEndRequest: protocol.AbstractGlobalEndRequest{
				Xid: transaction.Xid,
			},
		}
		res, err = getty.GetGettyRemotingClient().SendSyncRequest(req)
		if err != nil {
			log.Errorf("GlobalCommitRequest error, xid %s, error %v", transaction.Xid, err)
		} else {
			break
		}
	}
	if err == nil && res != nil {
		transaction.Status = res.(protocol.GlobalCommitResponse).GlobalStatus
	}
	model.UnbindXid(ctx)
	log.Infof("GlobalCommitRequest commit success, xid %s", transaction.Xid)
	return err
}

// Rollback the global transaction.
func (g *GlobalTransactionManager) Rollback(ctx context.Context, transaction *GlobalTransaction) error {
	if transaction.Role != model.LAUNCHER {
		log.Infof("Ignore Commit(): just involved in global transaction [{}]", transaction.Xid)
		return nil
	}
	if transaction.Xid == "" {
		return errors.New("Commit xid should not be empty")
	}

	// todo: replace retry with config
	var (
		err error
		res interface{}
	)
	for retry := 5; retry > 0; retry-- {
		req := protocol.GlobalRollbackRequest{
			AbstractGlobalEndRequest: protocol.AbstractGlobalEndRequest{
				Xid: transaction.Xid,
			},
		}
		res, err = getty.GetGettyRemotingClient().SendSyncRequest(req)
		if err != nil {
			log.Errorf("GlobalRollbackRequest error, xid %s, error %v", transaction.Xid, err)
		} else {
			break
		}
	}
	if err == nil && res != nil {
		transaction.Status = res.(protocol.GlobalRollbackResponse).GlobalStatus
	}
	model.UnbindXid(ctx)
	return err
}

// Suspend the global transaction.
func (g *GlobalTransactionManager) Suspend() (api.SuspendedResourcesHolder, error) {
	panic("implement me")
}

// Resume the global transaction.
func (g *GlobalTransactionManager) Resume(suspendedResourcesHolder api.SuspendedResourcesHolder) error {
	panic("implement me")
}

// report the global transaction status.
func (g *GlobalTransactionManager) GlobalReport(globalStatus model.GlobalStatus) error {
	panic("implement me")
}
