package state

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"reflect"
)

type TaskState interface {
	statelang.State

	CompensateState() string

	ForCompensation() bool

	ForUpdate() bool

	Retry() []Retry

	Catches() []ExceptionMatch

	Status() map[string]string

	Loop() Loop
}

type Loop interface {
	Parallel() int

	Collection() string

	ElementVariableName() string

	ElementIndexName() string

	CompletionCondition() string
}

type ExceptionMatch interface {
	Exceptions() []string
	// TODO: go dose not support get reflect.Type by string, not use it now
	ExceptionTypes() []reflect.Type

	SetExceptionTypes(ExceptionTypes []reflect.Type)

	Next() string
}

type Retry interface {
	Exceptions() []string

	IntervalSecond() float64

	MaxAttempt() int

	BackoffRate() float64
}

type ServiceTaskState interface {
	TaskState

	ServiceType() string

	ServiceName() string

	ServiceMethod() string

	ParameterTypes() []string

	Persist() bool

	RetryPersistModeUpdate() bool

	CompensatePersistModeUpdate() bool
}

type AbstractTaskState struct {
	*statelang.BaseState
	loop                        Loop
	catches                     []ExceptionMatch
	input                       []interface{}
	inputExpressions            []interface{}
	output                      map[string]interface{}
	outputExpressions           map[string]interface{}
	compensatePersistModeUpdate bool
	retryPersistModeUpdate      bool
	forCompensation             bool
	forUpdate                   bool
	persist                     bool
	compensateState             string
	status                      map[string]string
	retry                       []Retry
}

func NewAbstractTaskState() *AbstractTaskState {
	return &AbstractTaskState{
		BaseState: &statelang.BaseState{},
	}
}

func (a *AbstractTaskState) InputExpressions() []interface{} {
	return a.inputExpressions
}

func (a *AbstractTaskState) SetInputExpressions(inputExpressions []interface{}) {
	a.inputExpressions = inputExpressions
}

func (a *AbstractTaskState) OutputExpressions() map[string]interface{} {
	return a.outputExpressions
}

func (a *AbstractTaskState) SetOutputExpressions(outputExpressions map[string]interface{}) {
	a.outputExpressions = outputExpressions
}

func (a *AbstractTaskState) Input() []interface{} {
	return a.input
}

func (a *AbstractTaskState) SetInput(input []interface{}) {
	a.input = input
}

func (a *AbstractTaskState) Output() map[string]interface{} {
	return a.output
}

func (a *AbstractTaskState) SetOutput(output map[string]interface{}) {
	a.output = output
}

func (a *AbstractTaskState) CompensatePersistModeUpdate() bool {
	return a.compensatePersistModeUpdate
}

func (a *AbstractTaskState) SetCompensatePersistModeUpdate(isCompensatePersistModeUpdate bool) {
	a.compensatePersistModeUpdate = isCompensatePersistModeUpdate
}

func (a *AbstractTaskState) RetryPersistModeUpdate() bool {
	return a.retryPersistModeUpdate
}

func (a *AbstractTaskState) SetRetryPersistModeUpdate(retryPersistModeUpdate bool) {
	a.retryPersistModeUpdate = retryPersistModeUpdate
}

func (a *AbstractTaskState) Persist() bool {
	return a.persist
}

func (a *AbstractTaskState) SetPersist(persist bool) {
	a.persist = persist
}

func (a *AbstractTaskState) SetLoop(loop Loop) {
	a.loop = loop
}

func (a *AbstractTaskState) SetCatches(catches []ExceptionMatch) {
	a.catches = catches
}

func (a *AbstractTaskState) SetForCompensation(forCompensation bool) {
	a.forCompensation = forCompensation
}

func (a *AbstractTaskState) SetForUpdate(forUpdate bool) {
	a.forUpdate = forUpdate
}

func (a *AbstractTaskState) SetCompensateState(compensateState string) {
	a.compensateState = compensateState
}

func (a *AbstractTaskState) SetStatus(status map[string]string) {
	a.status = status
}

func (a *AbstractTaskState) SetRetry(retry []Retry) {
	a.retry = retry
}

func (a *AbstractTaskState) ForCompensation() bool {
	return a.forCompensation
}

func (a *AbstractTaskState) ForUpdate() bool {
	return a.forUpdate
}

func (a *AbstractTaskState) Catches() []ExceptionMatch {
	return a.catches
}

func (a *AbstractTaskState) Loop() Loop {
	return a.loop
}

func (a *AbstractTaskState) CompensateState() string {
	return a.compensateState
}

func (a *AbstractTaskState) Status() map[string]string {
	return a.status
}

func (a *AbstractTaskState) Retry() []Retry {
	return a.retry
}

type ServiceTaskStateImpl struct {
	*AbstractTaskState
	serviceType                 string
	serviceName                 string
	serviceMethod               string
	parameterTypes              []string
	method                      *reflect.Value
	persist                     bool
	retryPersistModeUpdate      bool
	compensatePersistModeUpdate bool
	isAsync                     bool
}

func NewServiceTaskStateImpl() *ServiceTaskStateImpl {
	return &ServiceTaskStateImpl{
		AbstractTaskState: NewAbstractTaskState(),
	}
}

func (s *ServiceTaskStateImpl) Method() *reflect.Value {
	return s.method
}

func (s *ServiceTaskStateImpl) SetMethod(method *reflect.Value) {
	s.method = method
}

func (s *ServiceTaskStateImpl) IsAsync() bool {
	return s.isAsync
}

func (s *ServiceTaskStateImpl) SetIsAsync(isAsync bool) {
	s.isAsync = isAsync
}

func (s *ServiceTaskStateImpl) SetServiceType(serviceType string) {
	s.serviceType = serviceType
}

func (s *ServiceTaskStateImpl) SetServiceName(serviceName string) {
	s.serviceName = serviceName
}

func (s *ServiceTaskStateImpl) SetServiceMethod(serviceMethod string) {
	s.serviceMethod = serviceMethod
}

func (s *ServiceTaskStateImpl) SetParameterTypes(parameterTypes []string) {
	s.parameterTypes = parameterTypes
}

func (s *ServiceTaskStateImpl) SetPersist(persist bool) {
	s.persist = persist
}

func (s *ServiceTaskStateImpl) SetRetryPersistModeUpdate(retryPersistModeUpdate bool) {
	s.retryPersistModeUpdate = retryPersistModeUpdate
}

func (s *ServiceTaskStateImpl) SetCompensatePersistModeUpdate(compensatePersistModeUpdate bool) {
	s.compensatePersistModeUpdate = compensatePersistModeUpdate
}

func (s *ServiceTaskStateImpl) Loop() Loop {
	return s.loop
}

func (s *ServiceTaskStateImpl) ServiceType() string {
	return s.serviceType
}

func (s *ServiceTaskStateImpl) ServiceName() string {
	return s.serviceName
}

func (s *ServiceTaskStateImpl) ServiceMethod() string {
	return s.serviceMethod
}

func (s *ServiceTaskStateImpl) ParameterTypes() []string {
	return s.parameterTypes
}

func (s *ServiceTaskStateImpl) Persist() bool {
	return s.persist
}

func (s *ServiceTaskStateImpl) RetryPersistModeUpdate() bool {
	return s.retryPersistModeUpdate
}

func (s *ServiceTaskStateImpl) CompensatePersistModeUpdate() bool {
	return s.compensatePersistModeUpdate
}

type LoopImpl struct {
	parallel            int
	collection          string
	elementVariableName string
	elementIndexName    string
	completionCondition string
}

func (l *LoopImpl) SetParallel(parallel int) {
	l.parallel = parallel
}

func (l *LoopImpl) SetCollection(collection string) {
	l.collection = collection
}

func (l *LoopImpl) SetElementVariableName(elementVariableName string) {
	l.elementVariableName = elementVariableName
}

func (l *LoopImpl) SetElementIndexName(elementIndexName string) {
	l.elementIndexName = elementIndexName
}

func (l *LoopImpl) SetCompletionCondition(completionCondition string) {
	l.completionCondition = completionCondition
}

func (l *LoopImpl) Parallel() int {
	return l.parallel
}

func (l *LoopImpl) Collection() string {
	return l.collection
}

func (l *LoopImpl) ElementVariableName() string {
	return l.elementVariableName
}

func (l *LoopImpl) ElementIndexName() string {
	return l.elementIndexName
}

func (l *LoopImpl) CompletionCondition() string {
	return l.completionCondition
}

type RetryImpl struct {
	exceptions     []string
	intervalSecond float64
	maxAttempt     int
	backoffRate    float64
}

func (r *RetryImpl) SetExceptions(exceptions []string) {
	r.exceptions = exceptions
}

func (r *RetryImpl) SetIntervalSecond(intervalSecond float64) {
	r.intervalSecond = intervalSecond
}

func (r *RetryImpl) SetMaxAttempt(maxAttempt int) {
	r.maxAttempt = maxAttempt
}

func (r *RetryImpl) SetBackoffRate(backoffRate float64) {
	r.backoffRate = backoffRate
}

func (r *RetryImpl) Exceptions() []string {
	return r.exceptions
}

func (r *RetryImpl) IntervalSecond() float64 {
	return r.intervalSecond
}

func (r *RetryImpl) MaxAttempt() int {
	return r.maxAttempt
}

func (r *RetryImpl) BackoffRate() float64 {
	return r.backoffRate
}

type ExceptionMatchImpl struct {
	exceptions     []string
	exceptionTypes []reflect.Type
	next           string
}

func (e *ExceptionMatchImpl) SetExceptions(errors []string) {
	e.exceptions = errors
}

func (e *ExceptionMatchImpl) SetNext(next string) {
	e.next = next
}

func (e *ExceptionMatchImpl) Exceptions() []string {
	return e.exceptions
}

func (e *ExceptionMatchImpl) ExceptionTypes() []reflect.Type {
	return e.exceptionTypes
}

func (e *ExceptionMatchImpl) SetExceptionTypes(exceptionTypes []reflect.Type) {
	e.exceptionTypes = exceptionTypes
}

func (e *ExceptionMatchImpl) Next() string {
	return e.next
}

type ScriptTaskState interface {
	TaskState

	ScriptType() string

	ScriptContent() string
}

type ScriptTaskStateImpl struct {
	*AbstractTaskState
	scriptType    string
	scriptContent string
}

func NewScriptTaskStateImpl() *ScriptTaskStateImpl {
	return &ScriptTaskStateImpl{
		AbstractTaskState: NewAbstractTaskState(),
		scriptType:        constant.DefaultScriptType,
	}
}

func (s *ScriptTaskStateImpl) SetScriptType(scriptType string) {
	s.scriptType = scriptType
}

func (s *ScriptTaskStateImpl) SetScriptContent(scriptContent string) {
	s.scriptContent = scriptContent
}

func (s *ScriptTaskStateImpl) ScriptType() string {
	return s.scriptType
}

func (s *ScriptTaskStateImpl) ScriptContent() string {
	return s.scriptContent
}
