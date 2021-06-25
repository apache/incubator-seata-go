package event

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
)

const (
	RoleTC = "tc"
	RoleTM = "client"
	RoleRM = "rm"
)

type GlobalTransactionEvent struct {
	id        int64
	role      string
	name      string
	beginTime int64
	endTime   int64
	status    apis.GlobalSession_GlobalStatus
}

func NewGlobalTransactionEvent(id int64, role string, name string, beginTime int64, endTime int64, status apis.GlobalSession_GlobalStatus) GlobalTransactionEvent {
	return GlobalTransactionEvent{
		id,
		role,
		name,
		beginTime,
		endTime,
		status,
	}
}

func (event GlobalTransactionEvent) GetID() int64 { return event.id }

func (event GlobalTransactionEvent) GetRole() string { return event.role }

func (event GlobalTransactionEvent) GetName() string { return event.name }

func (event GlobalTransactionEvent) GetBeginTime() int64 { return event.beginTime }

func (event GlobalTransactionEvent) GetEndTime() int64 { return event.endTime }

func (event GlobalTransactionEvent) GetStatus() apis.GlobalSession_GlobalStatus { return event.status }
