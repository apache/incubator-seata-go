package manager

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/transaction"
	"github.com/seata/seata-go/pkg/remoting/getty"
	"github.com/seata/seata-go/pkg/tm/api"
	"sync"
)

type GlobalTransaction struct {
	Xid    string
	Status transaction.GlobalStatus
	Role   transaction.GlobalTransactionRole
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
func (g *GlobalTransactionManager) Begin(ctx context.Context, gtr *GlobalTransaction, timeout int32, name string) error {
	if gtr.Role != transaction.LAUNCHER {
		log.Infof("Ignore Begin(): just involved in global transaction %s", gtr.Xid)
		return nil
	}
	if gtr.Xid != "" {
		return errors.New(fmt.Sprintf("Global transaction already exists,can't begin a new global transaction, currentXid = %s ", gtr.Xid))
	}

	req := message.GlobalBeginRequest{
		TransactionName: name,
		Timeout:         timeout,
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("GlobalBeginRequest error, xid %s, error %v", gtr.Xid, err)
		return err
	}
	if res == nil || res.(message.GlobalBeginResponse).ResultCode == message.ResultCodeFailed {
		log.Errorf("GlobalBeginRequest error, xid %s, res %v", gtr.Xid, res)
		return err
	}
	log.Infof("GlobalBeginRequest success, xid %s, res %v", gtr.Xid, res)

	gtr.Status = transaction.Begin
	gtr.Xid = res.(message.GlobalBeginResponse).Xid
	transaction.SetXID(ctx, res.(message.GlobalBeginResponse).Xid)
	return nil
}

// Commit the global transaction.
func (g *GlobalTransactionManager) Commit(ctx context.Context, gtr *GlobalTransaction) error {
	if gtr.Role != transaction.LAUNCHER {
		log.Infof("Ignore Commit(): just involved in global gtr [{}]", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return errors.New("Commit xid should not be empty")
	}

	// todo: replace retry with config
	var (
		err error
		res interface{}
	)
	for retry := 5; retry > 0; retry-- {
		req := message.GlobalCommitRequest{
			AbstractGlobalEndRequest: message.AbstractGlobalEndRequest{
				Xid: gtr.Xid,
			},
		}
		res, err = getty.GetGettyRemotingClient().SendSyncRequest(req)
		if err != nil {
			log.Errorf("GlobalCommitRequest error, xid %s, error %v", gtr.Xid, err)
		} else {
			break
		}
	}
	if err == nil && res != nil {
		gtr.Status = res.(message.GlobalCommitResponse).GlobalStatus
	}
	transaction.UnbindXid(ctx)
	log.Infof("GlobalCommitRequest commit success, xid %s", gtr.Xid)
	return err
}

// Rollback the global transaction.
func (g *GlobalTransactionManager) Rollback(ctx context.Context, gtr *GlobalTransaction) error {
	if gtr.Role != transaction.LAUNCHER {
		log.Infof("Ignore Commit(): just involved in global gtr [{}]", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return errors.New("Commit xid should not be empty")
	}

	// todo: replace retry with config
	var (
		err error
		res interface{}
	)
	for retry := 5; retry > 0; retry-- {
		req := message.GlobalRollbackRequest{
			AbstractGlobalEndRequest: message.AbstractGlobalEndRequest{
				Xid: gtr.Xid,
			},
		}
		res, err = getty.GetGettyRemotingClient().SendSyncRequest(req)
		if err != nil {
			log.Errorf("GlobalRollbackRequest error, xid %s, error %v", gtr.Xid, err)
		} else {
			break
		}
	}
	if err == nil && res != nil {
		gtr.Status = res.(message.GlobalRollbackResponse).GlobalStatus
	}
	transaction.UnbindXid(ctx)
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
func (g *GlobalTransactionManager) GlobalReport(globalStatus transaction.GlobalStatus) error {
	panic("implement me")
}
