package statelang

import "time"

type StateMachineImpl struct {
	id                          string
	tenantId                    string
	appName                     string
	name                        string
	comment                     string
	version                     string
	startState                  string
	status                      StateMachineStatus
	recoverStrategy             RecoverStrategy
	persist                     bool
	retryPersistModeUpdate      bool
	compensatePersistModeUpdate bool
	typeName                    string
	content                     string
	createTime                  time.Time
	states                      map[string]State
}

func NewStateMachineImpl() *StateMachineImpl {
	stateMap := make(map[string]State)
	return &StateMachineImpl{
		appName:  "SEATA",
		status:   Active,
		typeName: "STATE_LANG",
		states:   stateMap,
	}
}

func (s *StateMachineImpl) GetId() string {
	return s.id
}

func (s *StateMachineImpl) SetId(id string) {
	s.id = id
}

func (s *StateMachineImpl) GetName() string {
	return s.name
}

func (s *StateMachineImpl) SetName(name string) {
	s.name = name
}

func (s *StateMachineImpl) SetComment(comment string) {
	s.comment = comment
}

func (s *StateMachineImpl) GetComment() string {
	return s.comment
}

func (s *StateMachineImpl) GetStartState() string {
	return s.startState
}

func (s *StateMachineImpl) SetStartState(startState string) {
	s.startState = startState
}

func (s *StateMachineImpl) GetVersion() string {
	return s.version
}

func (s *StateMachineImpl) SetVersion(version string) {
	s.version = version
}

func (s *StateMachineImpl) GetStates() map[string]State {
	return s.states
}

func (s *StateMachineImpl) GetState(stateName string) State {
	if s.states == nil {
		return nil
	}

	return s.states[stateName]
}

func (s *StateMachineImpl) GetTenantId() string {
	return s.tenantId
}

func (s *StateMachineImpl) SetTenantId(tenantId string) {
	s.tenantId = tenantId
}

func (s *StateMachineImpl) GetAppName() string {
	return s.appName
}

func (s *StateMachineImpl) SetAppName(appName string) {
	s.appName = appName
}

func (s *StateMachineImpl) GetType() string {
	return s.typeName
}

func (s *StateMachineImpl) SetType(typeName string) {
	s.typeName = typeName
}

func (s *StateMachineImpl) GetStatus() StateMachineStatus {
	return s.status
}

func (s *StateMachineImpl) SetStatus(status StateMachineStatus) {
	s.status = status
}

func (s *StateMachineImpl) GetRecoverStrategy() RecoverStrategy {
	return s.recoverStrategy
}

func (s *StateMachineImpl) SetRecoverStrategy(recoverStrategy RecoverStrategy) {
	s.recoverStrategy = recoverStrategy
}

func (s *StateMachineImpl) IsPersist() bool {
	return s.persist
}

func (s *StateMachineImpl) IsRetryPersistModeUpdate() bool {
	return s.retryPersistModeUpdate
}

func (s *StateMachineImpl) isCompensatePersistModeUpdate() bool {
	return s.compensatePersistModeUpdate
}

func (s *StateMachineImpl) SetPersist(persist bool) {
	s.persist = persist
}

func (s *StateMachineImpl) GetContent() string {
	return s.content
}

func (s *StateMachineImpl) SetContent(content string) {
	s.content = content
}

func (s *StateMachineImpl) GetCreateTime() time.Time {
	return s.createTime
}

func (s *StateMachineImpl) SetCreateTime(createTime time.Time) {
	s.createTime = createTime
}
