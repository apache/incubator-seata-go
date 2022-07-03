package service

import (
	"context"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/tm"
)

type TestTCCServiceBusiness struct {
}

func (T TestTCCServiceBusiness) Prepare(ctx context.Context, params ...interface{}) error {
	log.Infof("TestTCCServiceBusiness Prepare, param %v", params)
	return nil
}

func (T TestTCCServiceBusiness) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Commit, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Rollback, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness) GetActionName() string {
	return "TestTCCServiceBusiness"
}

type TestTCCServiceBusiness2 struct {
}

func (T TestTCCServiceBusiness2) Prepare(ctx context.Context, params interface{}) error {
	log.Infof("TestTCCServiceBusiness2 Prepare, param %v", params)
	return nil
}

func (T TestTCCServiceBusiness2) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness2 Commit, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness2) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness2 Rollback, param %v", businessActionContext)
	return nil
}

func (T TestTCCServiceBusiness2) GetActionName() string {
	return "TestTCCServiceBusiness2"
}
