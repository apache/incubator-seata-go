package tcc

import (
	"context"
	"fmt"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/dubbo/client/service"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
)

var (
	names  []interface{}
	values = make([]reflect.Value, 0, 2)
)

func TestNewTCCServiceProxy(t *testing.T) {
	type args struct {
		service interface{}
	}

	userProvider := &service.UserProvider{}
	args1 := args{userProvider}
	args2 := args{userProvider}

	twoPhaseAction1, err1 := rm.ParseTwoPhaseAction(userProvider)
	twoPhaseAction2, err2 := rm.ParseTwoPhaseAction(userProvider)

	if err1 != nil {
		fmt.Println("current error ", err1)
	}

	if err2 != nil {
		fmt.Println("current error ", err2)
	}

	tests := []struct {
		name    string
		args    args
		want    *TCCServiceProxy
		wantErr assert.ErrorAssertionFunc
	}{
		{"test1", args1, &TCCServiceProxy{
			TCCResource: &TCCResource{
				ResourceGroupId: `default:"DEFAULT"`,
				AppName:         "seata-go-mock-app-name",
				TwoPhaseAction:  twoPhaseAction1}}, assert.NoError,
		},
		{"test2", args2, &TCCServiceProxy{
			TCCResource: &TCCResource{
				ResourceGroupId: `default:"DEFAULT"`,
				AppName:         "seata-go-mock-app-name",
				TwoPhaseAction:  twoPhaseAction2}}, assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTCCServiceProxy(tt.args.service)
			if !tt.wantErr(t, err, fmt.Sprintf("NewTCCServiceProxy(%v)", tt.args.service)) {
				return
			}
			assert.Equalf(t, tt.want, got, "NewTCCServiceProxy(%v)", tt.args.service)
		})
	}
}

func TestTCCServiceProxy_GetTransactionInfo(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}

	tcc := reflect.ValueOf(&rm.TwoPhaseAction{}).Elem()
	tcc.FieldByName("actionName").SetString("seataTwoPhaseName111")
	tcc.FieldByName("prepareMethodName").SetString("Prepare111")
	tcc.FieldByName("commitMethodName").SetString("Commit111")
	tcc.FieldByName("rollbackMethodName").SetString("Rollback11")
	twoParseAction := tcc.Interface()
	action := twoParseAction.(rm.TwoPhaseAction)

	tests := []struct {
		name   string
		fields fields
		want   tm.TransactionInfo
	}{
		{
			"test1", fields{referenceName: "test1", registerResourceOnce: sync.Once{},
				TCCResource: &TCCResource{ResourceGroupId: "default1", AppName: "app1",
					TwoPhaseAction: &action,
				},
			},
			tm.TransactionInfo{Name: "test-1", TimeOut: 111, Propagation: 111, LockRetryInternal: 222, LockRetryTimes: 222},
		},
		{
			"test2", fields{referenceName: "test1", registerResourceOnce: sync.Once{},
				TCCResource: &TCCResource{ResourceGroupId: "defaultw", AppName: "appw",
					TwoPhaseAction: &action,
				}},
			tm.TransactionInfo{Name: "test-2", TimeOut: 111, Propagation: 111, LockRetryInternal: 111, LockRetryTimes: 222},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TCCServiceProxy{
				referenceName:        tt.fields.referenceName,
				registerResourceOnce: tt.fields.registerResourceOnce,
				TCCResource:          tt.fields.TCCResource,
			}
			assert.Equalf(t1, tt.want, t.GetTransactionInfo(), "GetTransactionInfo()")
		})
	}
}

func TestTCCServiceProxy_Prepare(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}

	type args struct {
		ctx   context.Context
		param []interface{}
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr assert.ErrorAssertionFunc
	}{}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TCCServiceProxy{
				referenceName:        tt.fields.referenceName,
				registerResourceOnce: tt.fields.registerResourceOnce,
				TCCResource:          tt.fields.TCCResource,
			}
			got, err := t.Prepare(tt.args.ctx, tt.args.param...)
			if !tt.wantErr(t1, err, fmt.Sprintf("Prepare(%v, %v)", tt.args.ctx, tt.args.param)) {
				return
			}
			assert.Equalf(t1, tt.want, got, "Prepare(%v, %v)", tt.args.ctx, tt.args.param)
		})
	}
}

func TestTCCServiceProxy_Reference(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TCCServiceProxy{
				referenceName:        tt.fields.referenceName,
				registerResourceOnce: tt.fields.registerResourceOnce,
				TCCResource:          tt.fields.TCCResource,
			}
			assert.Equalf(t1, tt.want, t.Reference(), "Reference()")
		})
	}
}

func TestTCCServiceProxy_RegisterResource(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TCCServiceProxy{
				referenceName:        tt.fields.referenceName,
				registerResourceOnce: tt.fields.registerResourceOnce,
				TCCResource:          tt.fields.TCCResource,
			}
			tt.wantErr(t1, t.RegisterResource(), fmt.Sprintf("RegisterResource()"))
		})
	}
}

func TestTCCServiceProxy_SetReferenceName(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}
	type args struct {
		referenceName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TCCServiceProxy{
				referenceName:        tt.fields.referenceName,
				registerResourceOnce: tt.fields.registerResourceOnce,
				TCCResource:          tt.fields.TCCResource,
			}
			t.SetReferenceName(tt.args.referenceName)
		})
	}
}

func TestTCCServiceProxy_registeBranch(t1 *testing.T) {
	type fields struct {
		referenceName        string
		registerResourceOnce sync.Once
		TCCResource          *TCCResource
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TCCServiceProxy{
				referenceName:        tt.fields.referenceName,
				registerResourceOnce: tt.fields.registerResourceOnce,
				TCCResource:          tt.fields.TCCResource,
			}
			tt.wantErr(t1, t.registeBranch(tt.args.ctx), fmt.Sprintf("registeBranch(%v)", tt.args.ctx))
		})
	}
}

type TCCProvider struct {
}

func (t TCCProvider) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	return false, fmt.Errorf("execute two phase prepare method, param %v", params)
}

func (t *TCCProvider) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
}

func (t *TCCProvider) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return false, fmt.Errorf("execute two phase rollback method, xid %v", businessActionContext.Xid)
}

func (t *TCCProvider) GetActionName() string {
	return "TwoPhaseDemoService2"
}
