package event

import "github.com/dk-lockdown/seata-golang/base/meta"

const (
	RoleTC = "tc"
	RoleTM = "tm"
	RoleRM = "rm"
)

type GlobalTransactionEvent struct {
	id        int64
	role      string
	name      string
	beginTime int64
	endTime   int64
	status    meta.GlobalStatus
}

func NewGlobalTransactionEvent(id int64, role string, name string, beginTime int64, endTime int64, status meta.GlobalStatus) GlobalTransactionEvent {
	return GlobalTransactionEvent{
		id,
		role,
		name,
		beginTime,
		endTime,
		status,
	}
}

func (event GlobalTransactionEvent) GetId() int64 { return event.id }

func (event GlobalTransactionEvent) GetRole() string { return event.role }

func (event GlobalTransactionEvent) GetName() string { return event.name }

func (event GlobalTransactionEvent) GetBeginTime() int64 { return event.beginTime }

func (event GlobalTransactionEvent) GetEndTime() int64 { return event.endTime }

func (event GlobalTransactionEvent) GetStatus() meta.GlobalStatus { return event.status }
