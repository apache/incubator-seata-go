package rm

import (
	"context"
	"fmt"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseTwoPhaseActionGetMethodName(t *testing.T) {
	tests := []struct {
		service      interface{}
		wantService  *TwoPhaseAction
		wantHasError bool
		wantErrMsg   string
	}{{
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) error                             `seataTwoPhaseAction:"prepare"`
			Commit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"rollback" seataTwoPhaseServiceName:"seataTwoPhaseName"`
			GetName  func() string
		}{},
		wantService: &TwoPhaseAction{
			actionName:         "seataTwoPhaseName",
			prepareMethodName:  "Prepare",
			commitMethodName:   "Commit",
			rollbackMethodName: "Rollback",
		},
		wantHasError: false,
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) error                             `seataTwoPhaseAction:"prepare"`
			Commit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"rollback"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing two phase name",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) error                             `serviceName:"seataTwoPhaseName"`
			Commit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"rollback"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing prepare method",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) error `seataTwoPhaseAction:"commit" seataTwoPhaseName:"seataTwoPhaseName"`
			Commit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) error
			Rollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"rollback"`
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing prepare method",
	}, {
		service: &struct {
			Prepare  func(ctx context.Context, params interface{}) error                             `seataTwoPhaseAction:"prepare" seataTwoPhaseName:"seataTwoPhaseName"`
			Commit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"commit"`
			Rollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) error
			GetName  func() string
		}{},
		wantService:  nil,
		wantHasError: true,
		wantErrMsg:   "missing rollback method",
	}}

	for _, tt := range tests {
		actual, err := ParseTwoPhaseAction(tt.service)
		if tt.wantHasError {
			assert.NotNil(t, err)
			assert.Equal(t, tt.wantErrMsg, err.Error())
		} else {
			assert.Nil(t, err)
			assert.Equal(t, actual.actionName, tt.wantService.actionName)
			assert.Equal(t, actual.prepareMethodName, tt.wantService.prepareMethodName)
			assert.Equal(t, actual.commitMethodName, tt.wantService.commitMethodName)
			assert.Equal(t, actual.rollbackMethodName, tt.wantService.rollbackMethodName)
		}
	}
}

type TwoPhaseDemoService1 struct {
	TwoPhasePrepare  func(ctx context.Context, params ...interface{}) error                          `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
	TwoPhaseCommit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"commit"`
	TwoPhaseRollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"rollback"`
}

func NewTwoPhaseDemoService1() *TwoPhaseDemoService1 {
	return &TwoPhaseDemoService1{
		TwoPhasePrepare: func(ctx context.Context, params ...interface{}) error {
			return fmt.Errorf("execute two phase prepare method, param %v", params)
		},
		TwoPhaseCommit: func(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
			return fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
		},
		TwoPhaseRollback: func(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
			return fmt.Errorf("execute two phase rollback method, xid %v", businessActionContext.Xid)
		},
	}
}

func TestParseTwoPhaseActionExecuteMethod1(t *testing.T) {
	twoPhaseService, err := ParseTwoPhaseAction(NewTwoPhaseDemoService1())
	ctx := context.Background()
	assert.Nil(t, err)
	assert.Equal(t, "TwoPhasePrepare", twoPhaseService.prepareMethodName)
	assert.Equal(t, "TwoPhaseCommit", twoPhaseService.commitMethodName)
	assert.Equal(t, "TwoPhaseRollback", twoPhaseService.rollbackMethodName)
	assert.Equal(t, "TwoPhaseDemoService", twoPhaseService.twoPhaseService)
	assert.Equal(t, "execute two phase prepare method, param [[11]]", twoPhaseService.Prepare(ctx, 11).Error())
	assert.Equal(t, "execute two phase commit method, xid 1234", twoPhaseService.Commit(ctx, tm.BusinessActionContext{Xid: "1234"}).Error())
	assert.Equal(t, "execute two phase rollback method, xid 1234", twoPhaseService.Rollback(ctx, tm.BusinessActionContext{Xid: "1234"}).Error())
	assert.Equal(t, "TwoPhaseDemoService", twoPhaseService.GetActionName())
}

type TwoPhaseDemoService2 struct {
}

func (t *TwoPhaseDemoService2) Prepare(ctx context.Context, params ...interface{}) error {
	return fmt.Errorf("execute two phase prepare method, param %v", params)
}

func (t *TwoPhaseDemoService2) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	return fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
}

func (t *TwoPhaseDemoService2) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	return fmt.Errorf("execute two phase rollback method, xid %v", businessActionContext.Xid)
}

func (t *TwoPhaseDemoService2) GetActionName() string {
	return "TwoPhaseDemoService2"
}

func TestParseTwoPhaseActionExecuteMethod2(t *testing.T) {
	twoPhaseService, err := ParseTwoPhaseAction(&TwoPhaseDemoService2{})
	ctx := context.Background()
	assert.Nil(t, err)
	assert.Equal(t, "execute two phase prepare method, param [[11]]", twoPhaseService.Prepare(ctx, 11).Error())
	assert.Equal(t, "execute two phase commit method, xid 1234", twoPhaseService.Commit(ctx, tm.BusinessActionContext{Xid: "1234"}).Error())
	assert.Equal(t, "execute two phase rollback method, xid 1234", twoPhaseService.Rollback(ctx, tm.BusinessActionContext{Xid: "1234"}).Error())
	assert.Equal(t, "TwoPhaseDemoService2", twoPhaseService.GetActionName())
}
