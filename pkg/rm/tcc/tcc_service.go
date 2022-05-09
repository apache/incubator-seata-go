package tcc

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/rm/api"
)

type TCCService interface {
	Prepare(ctx context.Context, param interface{}) error
	Commit(ctx context.Context, businessActionContext api.BusinessActionContext) error
	Rollback(ctx context.Context, businessActionContext api.BusinessActionContext) error
}
