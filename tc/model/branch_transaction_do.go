package model

import "time"

type BranchTransactionDO struct {
	Xid string

	TransactionId int64

	BranchId int64

	ResourceGroupId string

	ResourceId string

	BranchType string

	Status int32

	ClientId string

	ApplicationData []byte

	GmtCreate time.Time

	GmtModified time.Time
}
