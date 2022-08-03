package testdata

import (
	"reflect"

	"github.com/agiledragon/gomonkey"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
)

var (
	rmRemoting   *rm.RMRemoting
	TestBranchId = int64(1234567890)
)

func init() {
	initBranchRegister()
}

func initBranchRegister() {
	var registerResource = func(branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
		return TestBranchId, nil
	}
	gomonkey.ApplyMethod(reflect.TypeOf(rmRemoting), "BranchRegister", registerResource)
}
