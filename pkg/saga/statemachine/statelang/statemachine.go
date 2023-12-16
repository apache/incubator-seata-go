package statelang

import (
	"time"
)

type StateMachineStatus string

const (
	Active   StateMachineStatus = "Active"
	Inactive StateMachineStatus = "Inactive"
)

// RecoverStrategy : Recover Strategy
type RecoverStrategy string

const (
	//Compensate stateMachine
	Compensate RecoverStrategy = "Compensate"
	// Forward  stateMachine
	Forward RecoverStrategy = "Forward"
)

func ValueOf(recoverStrategy string) (RecoverStrategy, bool) {
	switch recoverStrategy {
	case "Compensate":
		return Compensate, true
	case "Forward":
		return Forward, true
	default:
		var recoverStrategy RecoverStrategy
		return recoverStrategy, false
	}
}

type StateMachine interface {
	GetId() string

	SetId(id string)

	GetName() string

	SetName(name string)

	GetComment() string

	SetComment(comment string)

	GetStartState() string

	SetStartState(startState string)

	GetVersion() string

	SetVersion(version string)

	GetStates() map[string]State

	GetState(stateName string) State

	GetTenantId() string

	SetTenantId(tenantId string)

	GetAppName() string

	SetAppName(appName string)

	GetType() string

	SetType(typeName string)

	GetStatus() StateMachineStatus

	SetStatus(status StateMachineStatus)

	GetRecoverStrategy() RecoverStrategy

	SetRecoverStrategy(recoverStrategy RecoverStrategy)

	IsPersist() bool

	IsRetryPersistModeUpdate() bool

	isCompensatePersistModeUpdate() bool

	SetPersist(persist bool)

	GetContent() string

	SetContent(content string)

	GetCreateTime() time.Time

	SetCreateTime(createTime time.Time)
}

type State interface {
	GetName() string

	SetName(name string)

	GetComment() string

	SetComment(comment string)

	GetType() string

	SetType(typeName string)

	GetNext() string

	SetNext(next string)

	GetStateMachine() StateMachine

	SetStateMachine(machine StateMachine)
}

type BaseState struct {
	name         string
	comment      string
	typeName     string
	next         string
	stateMachine StateMachine
}

func (b *BaseState) GetName() string {
	return b.name
}

func (b *BaseState) SetName(name string) {
	b.name = name
}

func (b *BaseState) GetComment() string {
	return b.comment
}

func (b *BaseState) SetComment(comment string) {
	b.comment = comment
}

func (b *BaseState) GetType() string {
	return b.typeName
}

func (b *BaseState) SetType(typeName string) {
	b.typeName = typeName
}

func (b *BaseState) GetNext() string {
	return b.next
}

func (b *BaseState) SetNext(next string) {
	b.next = next
}

func (b *BaseState) GetStateMachine() StateMachine {
	return b.stateMachine
}

func (b *BaseState) SetStateMachine(machine StateMachine) {
	b.stateMachine = machine
}
