package tcc

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/dubbo/client/service"

	"github.com/stretchr/testify/assert"
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

	twoPhaseAction1, _ := rm.ParseTwoPhaseAction(userProvider)
	twoPhaseAction2, _ := rm.ParseTwoPhaseAction(userProvider)

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

	userProvider := &service.UserProvider{}
	twoPhaseAction1, _ := rm.ParseTwoPhaseAction(userProvider)

	tests := []struct {
		name   string
		fields fields
		want   tm.TransactionInfo
	}{
		{
			"test1", fields{referenceName: "test1", registerResourceOnce: sync.Once{},
				TCCResource: &TCCResource{ResourceGroupId: "default1", AppName: "app1",
					TwoPhaseAction: twoPhaseAction1,
				},
			},
			tm.TransactionInfo{Name: "TwoPhaseDemoService", TimeOut: 10000, Propagation: 0, LockRetryInternal: 0, LockRetryTimes: 0},
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
