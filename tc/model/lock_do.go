package model

import "time"

type LockDO struct {
	Xid string `xorm:"xid"`

	TransactionId int64 `xorm:"transaction_id"`

	BranchId int64 `xorm:"branch_id"`

	ResourceId string `xorm:"resource_id"`

	TableName string `xorm:"table_name"`

	Pk string `xorm:"pk"`

	RowKey string `xorm:"row_key"`

	GmtCreate time.Time `xorm:"created"`

	GmtModified time.Time `xorm:"updated"`
}
