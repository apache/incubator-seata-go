package main

import (
	"context"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/tm"
)

type RMService struct {
}

func (b *RMService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("TRMService Prepare, param %v", params)
	return true, nil
}

func (b *RMService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("RMService Commit, param %v", businessActionContext)
	return true, nil
}

func (b *RMService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("RMService Rollback, param %v", businessActionContext)
	return true, nil
}

func (b *RMService) GetActionName() string {
	return "ginTccRMService"
}
