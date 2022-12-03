package conn

import "time"

type XAResource interface {
	Commit(xid string, onePhase bool) error
	End(xid string, flags int) error
	Forget(xid string) error
	GetTransactionTimeout() time.Duration
	IsSameRM(resource XAResource) bool
	XAPrepare(xid string) (int, error)
	Recover(flag int) []string
	Rollback(xid string) error
	SetTransactionTimeout(duration time.Duration) bool
	Start(xid string, flags int) error
}
