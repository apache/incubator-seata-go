package model

import "github.com/dk-lockdown/seata-golang/base/meta"

type SessionCondition struct {
	TransactionId      int64
	Xid                string
	Status             meta.GlobalStatus
	Statuses           []meta.GlobalStatus
	OverTimeAliveMills int64
}
