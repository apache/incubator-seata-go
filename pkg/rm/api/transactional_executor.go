package api

import "context"

type TransactionalExecutor interface {
	Execute(ctx context.Context, param interface{}) (interface{}, error)
	GetTransactionInfo() TransactionInfo
}
