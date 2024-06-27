package statelang

import (
	"sync"
	"time"
)

type ExecutionStatus string

const (
	// RU Running
	RU ExecutionStatus = "RU"
	// SU Succeed
	SU ExecutionStatus = "SU"
	// FA Failed
	FA ExecutionStatus = "FA"
	// UN Unknown
	UN ExecutionStatus = "UN"
	// SK Skipped
	SK ExecutionStatus = "SK"
)

type StateMachineInstance interface {
	ID() string

	SetID(id string)

	MachineID() string

	SetMachineID(machineID string)

	TenantID() string

	SetTenantID(tenantID string)

	ParentID() string

	SetParentID(parentID string)

	StartedTime() time.Time

	SetStartedTime(startedTime time.Time)

	EndTime() time.Time

	SetEndTime(endTime time.Time)

	StateList() []StateInstance

	State(stateId string) StateInstance

	PutState(stateId string, stateInstance StateInstance)

	Status() ExecutionStatus

	SetStatus(status ExecutionStatus)

	CompensationStatus() ExecutionStatus

	SetCompensationStatus(compensationStatus ExecutionStatus)

	IsRunning() bool

	SetRunning(isRunning bool)

	UpdatedTime() time.Time

	SetUpdatedTime(updatedTime time.Time)

	BusinessKey() string

	SetBusinessKey(businessKey string)

	Exception() error

	SetException(err error)

	StartParams() map[string]interface{}

	SetStartParams(startParams map[string]interface{})

	EndParams() map[string]interface{}

	SetEndParams(endParams map[string]interface{})

	Context() map[string]interface{}

	PutContext(key string, value interface{})

	SetContext(context map[string]interface{})

	StateMachine() StateMachine

	SetStateMachine(stateMachine StateMachine)

	SerializedStartParams() interface{}

	SetSerializedStartParams(serializedStartParams interface{})

	SerializedEndParams() interface{}

	SetSerializedEndParams(serializedEndParams interface{})

	SerializedError() interface{}

	SetSerializedError(serializedError interface{})
}

type StateMachineInstanceImpl struct {
	id                    string
	machineId             string
	tenantId              string
	parentId              string
	businessKey           string
	startParams           map[string]interface{}
	serializedStartParams interface{}
	startedTime           time.Time
	endTime               time.Time
	updatedTime           time.Time
	exception             error
	serializedError       interface{}
	endParams             map[string]interface{}
	serializedEndParams   interface{}
	status                ExecutionStatus
	compensationStatus    ExecutionStatus
	isRunning             bool
	context               map[string]interface{}
	stateMachine          StateMachine
	stateList             []StateInstance
	stateMap              map[string]StateInstance

	contextMutex sync.RWMutex // Mutex to protect concurrent access to context
	stateMutex   sync.RWMutex // Mutex to protect concurrent access to stateList and stateMap
}

func NewStateMachineInstanceImpl() *StateMachineInstanceImpl {
	return &StateMachineInstanceImpl{
		startParams: make(map[string]interface{}),
		endParams:   make(map[string]interface{}),
		stateList:   make([]StateInstance, 0),
		stateMap:    make(map[string]StateInstance)}
}

func (s *StateMachineInstanceImpl) ID() string {
	return s.id
}

func (s *StateMachineInstanceImpl) SetID(id string) {
	s.id = id
}

func (s *StateMachineInstanceImpl) MachineID() string {
	return s.machineId
}

func (s *StateMachineInstanceImpl) SetMachineID(machineID string) {
	s.machineId = machineID
}

func (s *StateMachineInstanceImpl) TenantID() string {
	return s.tenantId
}

func (s *StateMachineInstanceImpl) SetTenantID(tenantID string) {
	s.tenantId = tenantID
}

func (s *StateMachineInstanceImpl) ParentID() string {
	return s.parentId
}

func (s *StateMachineInstanceImpl) SetParentID(parentID string) {
	s.parentId = parentID
}

func (s *StateMachineInstanceImpl) StartedTime() time.Time {
	return s.startedTime
}

func (s *StateMachineInstanceImpl) SetStartedTime(startedTime time.Time) {
	s.startedTime = startedTime
}

func (s *StateMachineInstanceImpl) EndTime() time.Time {
	return s.endTime
}

func (s *StateMachineInstanceImpl) SetEndTime(endTime time.Time) {
	s.endTime = endTime
}

func (s *StateMachineInstanceImpl) StateList() []StateInstance {
	return s.stateList
}

func (s *StateMachineInstanceImpl) State(stateId string) StateInstance {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()

	return s.stateMap[stateId]
}

func (s *StateMachineInstanceImpl) PutState(stateId string, stateInstance StateInstance) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	stateInstance.SetStateMachineInstance(s)
	s.stateMap[stateId] = stateInstance
	s.stateList = append(s.stateList, stateInstance)
}

func (s *StateMachineInstanceImpl) Status() ExecutionStatus {
	return s.status
}

func (s *StateMachineInstanceImpl) SetStatus(status ExecutionStatus) {
	s.status = status
}

func (s *StateMachineInstanceImpl) CompensationStatus() ExecutionStatus {
	return s.compensationStatus
}

func (s *StateMachineInstanceImpl) SetCompensationStatus(compensationStatus ExecutionStatus) {
	s.compensationStatus = compensationStatus
}

func (s *StateMachineInstanceImpl) IsRunning() bool {
	return s.isRunning
}

func (s *StateMachineInstanceImpl) SetRunning(isRunning bool) {
	s.isRunning = isRunning
}

func (s *StateMachineInstanceImpl) UpdatedTime() time.Time {
	return s.updatedTime
}

func (s *StateMachineInstanceImpl) SetUpdatedTime(updatedTime time.Time) {
	s.updatedTime = updatedTime
}

func (s *StateMachineInstanceImpl) BusinessKey() string {
	return s.businessKey
}

func (s *StateMachineInstanceImpl) SetBusinessKey(businessKey string) {
	s.businessKey = businessKey
}

func (s *StateMachineInstanceImpl) Exception() error {
	return s.exception
}

func (s *StateMachineInstanceImpl) SetException(err error) {
	s.exception = err
}

func (s *StateMachineInstanceImpl) StartParams() map[string]interface{} {
	return s.startParams
}

func (s *StateMachineInstanceImpl) SetStartParams(startParams map[string]interface{}) {
	s.startParams = startParams
}

func (s *StateMachineInstanceImpl) EndParams() map[string]interface{} {
	return s.endParams
}

func (s *StateMachineInstanceImpl) SetEndParams(endParams map[string]interface{}) {
	s.endParams = endParams
}

func (s *StateMachineInstanceImpl) Context() map[string]interface{} {
	return s.context
}

func (s *StateMachineInstanceImpl) PutContext(key string, value interface{}) {
	s.contextMutex.Lock()
	defer s.contextMutex.Unlock()

	s.context[key] = value
}

func (s *StateMachineInstanceImpl) SetContext(context map[string]interface{}) {
	s.context = context
}

func (s *StateMachineInstanceImpl) StateMachine() StateMachine {
	return s.stateMachine
}

func (s *StateMachineInstanceImpl) SetStateMachine(stateMachine StateMachine) {
	s.stateMachine = stateMachine
	s.machineId = stateMachine.ID()
}

func (s *StateMachineInstanceImpl) SerializedStartParams() interface{} {
	return s.serializedStartParams
}

func (s *StateMachineInstanceImpl) SetSerializedStartParams(serializedStartParams interface{}) {
	s.serializedStartParams = serializedStartParams
}

func (s *StateMachineInstanceImpl) SerializedEndParams() interface{} {
	return s.serializedEndParams
}

func (s *StateMachineInstanceImpl) SetSerializedEndParams(serializedEndParams interface{}) {
	s.serializedEndParams = serializedEndParams
}

func (s *StateMachineInstanceImpl) SerializedError() interface{} {
	return s.serializedError
}

func (s *StateMachineInstanceImpl) SetSerializedError(serializedError interface{}) {
	s.serializedError = serializedError
}
