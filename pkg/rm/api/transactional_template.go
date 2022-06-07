package api

import (
	"context"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/utils/log"
	"sync"
)

var (
	// singletone ResourceManagerFacade
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
	if !model.IsSeataContext(ctx) {
		err := errors.New("context should be inited as seata context!")
		log.Error(err)
		return nil, err
	}

	if model.GetTransactionRole(ctx) == nil {
		model.SetTransactionRole(ctx, model.LAUNCHER)
	}

	var tx *tm.GlobalTransaction
	if model.HasXID(ctx) {
		tx = &tm.GlobalTransaction{
			Xid:    model.GetXID(ctx),
			Status: model.Begin,
			Role:   model.PARTICIPANT,
		}
	}

	// todo: Handle the transaction propagation.

	if tx == nil {
		tx = &tm.GlobalTransaction{
			Xid:    model.GetXID(ctx),
			Status: model.UnKnown,
			Role:   model.LAUNCHER,
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
		return nil, tm.GetGlobalTransactionManager().Rollback(ctx, tx)
	}

	// commit global transaction
	err = t.CommitTransaction(ctx, tx)
	if err != nil {
		log.Infof("transactionTemplate: commit transaction failed, error %v", err)
		// rollback transaction
		return nil, tm.GetGlobalTransactionManager().Rollback(ctx, tx)
	}
	return res, err
}

func (TransactionTemplate) BeginTransaction(ctx context.Context, tx *tm.GlobalTransaction, timeout int32, name string) error {
	return tm.GetGlobalTransactionManager().Begin(ctx, tx, timeout, name)
}

func (TransactionTemplate) CommitTransaction(ctx context.Context, tx *tm.GlobalTransaction) error {
	return tm.GetGlobalTransactionManager().Commit(ctx, tx)
}
