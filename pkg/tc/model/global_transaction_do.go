package model

import (
	"time"
)

// GlobalTransactionDO for persist GlobalTransaction.
type GlobalTransactionDO struct {
	XID string `xorm:"xid"`

	TransactionID int64 `xorm:"transaction_id"`

	Status int32 `xorm:"status"`

	ApplicationID string `xorm:"application_id"`

	TransactionServiceGroup string `xorm:"transaction_service_group"`

	TransactionName string `xorm:"transaction_name"`

	Timeout int32 `xorm:"timeout"`

	BeginTime int64 `xorm:"begin_time"`

	ApplicationData []byte `xorm:"application_data"`

	GmtCreate time.Time `xorm:"gmt_create"`

	GmtModified time.Time `xorm:"gmt_modified"`
}
