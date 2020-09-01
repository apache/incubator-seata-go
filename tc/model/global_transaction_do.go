package model

import "time"

// GlobalTransactionDO for persist GlobalTransaction.
type GlobalTransactionDO struct {
	Xid string `xorm:"xid"`

	TransactionId int64 `xorm:"transaction_id"`

	Status int32 `xorm:"status"`

	ApplicationId string `xorm:"application_id"`

	TransactionServiceGroup string `xorm:"transaction_service_group"`

	TransactionName string `xorm:"transaction_name"`

	Timeout int32 `xorm:"timeout"`

	BeginTime int64 `xorm:"begin_time"`

	ApplicationData []byte `xorm:"application_data"`

	GmtCreate time.Time `xorm:"gmt_create"`

	GmtModified time.Time `xorm:"gmt_modified"`
}
