package testdata

import (
	"context"
	"github.com/seata/seata-go/pkg/tm"
)

type TestSagaTwoPhaseService struct{}

func (*TestSagaTwoPhaseService) Action(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, nil
}

func (*TestSagaTwoPhaseService) Compensation(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, nil
}
