package model

import (
	"time"
)

// BranchTransactionDO for persist BranchTransaction.
type BranchTransactionDO struct {
	XID string `xorm:"xid"`

	TransactionID int64 `xorm:"transaction_id"`

	BranchID int64 `xorm:"branch_id"`

	ResourceGroupID string `xorm:"resource_group_id"`

	ResourceID string `xorm:"resource_id"`

	BranchType string `xorm:"branch_type"`

	Status int32 `xorm:"status"`

	ClientID string `xorm:"client_id"`

	ApplicationData []byte `xorm:"application_data"`

	GmtCreate time.Time `xorm:"gmt_create"`

	GmtModified time.Time `xorm:"gmt_modified"`
}
