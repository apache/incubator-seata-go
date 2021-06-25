package server

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"time"
)

type GlobalSessionLocker interface {
	TryLock(session *apis.GlobalSession, timeout time.Duration) (bool, error)

	Unlock(session *apis.GlobalSession)
}

type UnimplementedGlobalSessionLocker struct {
}

func (locker *UnimplementedGlobalSessionLocker) TryLock(session *apis.GlobalSession, timeout time.Duration) (bool, error) {
	return true, nil
}

func (locker *UnimplementedGlobalSessionLocker) Unlock(session *apis.GlobalSession) {

}
