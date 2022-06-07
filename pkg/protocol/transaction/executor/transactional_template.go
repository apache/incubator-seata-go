package executor

import (
	"context"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/transaction"
	"github.com/seata/seata-go/pkg/protocol/transaction/manager"
	"sync"
)

type TransactionalExecutor interface {
	Execute(ctx context.Context, param interface{}) (interface{}, error)
	GetTransactionInfo() transaction.TransactionInfo
}

var (
	transactionTemplate     *TransactionTemplate
	onceTransactionTemplate = &sync.Once{}
)

func GetTransactionTemplate() *TransactionTemplate {
	if transactionTemplate == nil {
		onceTransactionTemplate.Do(func() {
			transactionTemplate = &TransactionTemplate{}
		})
	}
	return transactionTemplate
}

type TransactionTemplate struct {
}

func (t *TransactionTemplate) Execute(ctx context.Context, business TransactionalExecutor, param interface{}) (interface{}, error) {
	if !transaction.IsSeataContext(ctx) {
		err := errors.New("context should be inited as seata context!")
		log.Error(err)
		return nil, err
	}

	if transaction.GetTransactionRole(ctx) == nil {
		transaction.SetTransactionRole(ctx, transaction.LAUNCHER)
	}

	var tx *manager.GlobalTransaction
	if transaction.HasXID(ctx) {
		tx = &manager.GlobalTransaction{
			Xid:    transaction.GetXID(ctx),
			Status: transaction.Begin,
			Role:   transaction.PARTICIPANT,
		}
	}

	// todo: Handle the transaction propagation.

	if tx == nil {
		tx = &manager.GlobalTransaction{
			Xid:    transaction.GetXID(ctx),
			Status: transaction.UnKnown,
			Role:   transaction.LAUNCHER,
		}
	}

	// todo: set current tx config to holder

	// begin global transaction
	err := t.BeginTransaction(ctx, tx, business.GetTransactionInfo().TimeOut, business.GetTransactionInfo().Name)
	if err != nil {
		log.Infof("transactionTemplate: begin transaction failed, error %v", err)
		return nil, err
	}

	// do your business
	res, err := business.Execute(ctx, param)
	if err != nil {
		log.Infof("transactionTemplate: execute business failed, error %v", err)
		return nil, manager.GetGlobalTransactionManager().Rollback(ctx, tx)
	}

	// commit global transaction
	err = t.CommitTransaction(ctx, tx)
	if err != nil {
		log.Infof("transactionTemplate: commit transaction failed, error %v", err)
		// rollback transaction
		return nil, manager.GetGlobalTransactionManager().Rollback(ctx, tx)
	}
	return res, err
}

func (TransactionTemplate) BeginTransaction(ctx context.Context, tx *manager.GlobalTransaction, timeout int32, name string) error {
	return manager.GetGlobalTransactionManager().Begin(ctx, tx, timeout, name)
}

func (TransactionTemplate) CommitTransaction(ctx context.Context, tx *manager.GlobalTransaction) error {
	return manager.GetGlobalTransactionManager().Commit(ctx, tx)
}
