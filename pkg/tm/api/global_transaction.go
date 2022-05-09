package api

import (
	"github.com/seata/seata-go/pkg/model"
)

type GlobalTransactionRole int8

const (
	LAUNCHER    GlobalTransactionRole = 0
	PARTICIPANT GlobalTransactionRole = 1
)

type GlobalTransaction interface {

	// Begin a new global transaction with given timeout and given name.
	begin(timeout int64, name string) error

	// Commit the global transaction.
	commit() error

	// Rollback the global transaction.
	rollback() error

	// Suspend the global transaction.
	suspend() (SuspendedResourcesHolder, error)

	// Resume the global transaction.
	resume(suspendedResourcesHolder SuspendedResourcesHolder) error

	// Ask TC for current status of the corresponding global transaction.
	getStatus() (model.GlobalStatus, error)

	// Get XID.
	getXid() string

	// report the global transaction status.
	globalReport(globalStatus model.GlobalStatus) error

	// local status of the global transaction.
	getLocalStatus() model.GlobalStatus

	// get global transaction role.
	getGlobalTransactionRole() GlobalTransactionRole
}
