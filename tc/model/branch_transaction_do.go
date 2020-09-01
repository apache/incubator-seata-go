package model

import "time"

// BranchTransactionDO for persist BranchTransaction.
type BranchTransactionDO struct {
	Xid string `xorm:"xid"`

	TransactionId int64 `xorm:"transaction_id"`

	BranchId int64 `xorm:"branch_id"`

	ResourceGroupId string `xorm:"resource_group_id"`

	ResourceId string `xorm:"resource_id"`

	BranchType string `xorm:"branch_type"`

	Status int32 `xorm:"status"`

	ClientId string `xorm:"client_id"`

	ApplicationData []byte `xorm:"application_data"`

	GmtCreate time.Time `xorm:"gmt_create"`

	GmtModified time.Time `xorm:"gmt_modified"`
}
