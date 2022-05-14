package tcc

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/rm/api"
	"github.com/seata/seata-go/pkg/rm/tcc/remoting"
)

type TCCService interface {
	Prepare(ctx context.Context, params interface{}) error
	Commit(ctx context.Context, businessActionContext api.BusinessActionContext) error
	Rollback(ctx context.Context, businessActionContext api.BusinessActionContext) error

	GetActionName() string
	GetRemoteType() remoting.RemoteType
	GetServiceType() remoting.ServiceType
}
