package model

import (
	"time"
)

// LockDO for persist Lock.
type LockDO struct {
	Xid string `xorm:"xid"`

	TransactionID int64 `xorm:"transaction_id"`

	BranchID int64 `xorm:"branch_id"`

	ResourceID string `xorm:"resource_id"`

	TableName string `xorm:"table_name"`

	Pk string `xorm:"pk"`

	RowKey string `xorm:"row_key"`

	GmtCreate time.Time `xorm:"created"`

	GmtModified time.Time `xorm:"updated"`
}
