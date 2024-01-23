package saga

import (
	"context"
	"github.com/agiledragon/gomonkey"
	gostnet "github.com/dubbogo/gost/net"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	testdata2 "github.com/seata/seata-go/testdata"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"
	"time"
)

var (
	testSagaServiceProxy *SagaServiceProxy
	testBranchID         = int64(121324345353)
	names                []interface{}
	values               = make([]reflect.Value, 0, 2)
)

type UserProvider struct {
	Prepare       func(ctx context.Context, params ...interface{}) (bool, error)                           `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
	Commit        func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
	Rollback      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
	GetActionName func() string
}

func InitMock() {
	log.Init()
	var (
		registerResource = func(_ *SagaServiceProxy) error {
			return nil
		}
		branchRegister = func(_ *rm.RMRemoting, param rm.BranchRegisterParam) (int64, error) {
			return testBranchID, nil
		}
	)
	log.Infof("run init mock")
	gomonkey.ApplyMethod(reflect.TypeOf(testSagaServiceProxy), "RegisterResource", registerResource)
	gomonkey.ApplyMethod(reflect.TypeOf(rm.GetRMRemotingInstance()), "BranchRegister", branchRegister)
	testSagaServiceProxy, _ = NewSagaServiceProxy(GetTestTwoPhaseService())
}

func TestMain(m *testing.M) {
	InitMock()
	code := m.Run()
	os.Exit(code)
}

func TestInitActionContext(t *testing.T) {
	param := struct {
		name  string `tccParam:"name"`
		Age   int64  `tccParam:""`
		Addr  string `tccParam:"addr"`
		Job   string `tccParam:"-"`
		Class string
		Other []int8 `tccParam:"Other"`
	}{
		name:  "Jack",
		Age:   20,
		Addr:  "Earth",
		Job:   "Dor",
		Class: "1-2",
		Other: []int8{1, 2, 3},
	}

	now := time.Now()
	p := gomonkey.ApplyFunc(time.Now, func() time.Time {
		return now
	})
	defer p.Reset()
	result := testSagaServiceProxy.initActionContext(param)
	localIp, _ := gostnet.GetLocalIP()
	assert.Equal(t, map[string]interface{}{
		"addr":                   "Earth",
		"Other":                  []int8{1, 2, 3},
		constant.ActionStartTime: now.UnixNano() / 1e6,
		constant.PrepareMethod:   "Prepare",
		constant.CommitMethod:    "Commit",
		constant.RollbackMethod:  "Rollback",
		constant.ActionName:      testdata2.ActionName,
		constant.HostName:        localIp,
	}, result)
}

func GetTestTwoPhaseService() rm.SagaActionInterface {
	return &testdata2.TestSagaTwoPhaseService{}
}
