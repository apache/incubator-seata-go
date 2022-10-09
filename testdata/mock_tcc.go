package testdata

import (
	"context"

	"github.com/seata/seata-go/pkg/tm"
)

const (
	ActionName = "TestActionName"
)

type TestTwoPhaseService struct{}

func (*TestTwoPhaseService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	return true, nil
}

func (*TestTwoPhaseService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, nil
}

func (*TestTwoPhaseService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, nil
}

func (*TestTwoPhaseService) GetActionName() string {
	return ActionName
}
