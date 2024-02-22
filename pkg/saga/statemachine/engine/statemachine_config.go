package engine

import (
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/events"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/expr"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/invoker"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/sequence"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/status_decision"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/store"
)

type StateMachineConfig interface {
	StateLogRepository() store.StateLogRepository

	StateMachineRepository() store.StateMachineRepository

	StateLogStore() store.StateLogStore

	StateLangStore() store.StateLangStore

	ExpressionFactoryManager() expr.ExpressionFactoryManager

	ExpressionResolver() expr.ExpressionResolver

	SeqGenerator() sequence.SeqGenerator

	StatusDecisionStrategy() status_decision.StatusDecisionStrategy

	EventPublisher() events.EventPublisher

	AsyncEventPublisher() events.EventPublisher

	ServiceInvokerManager() invoker.ServiceInvokerManager

	ScriptInvokerManager() invoker.ScriptInvokerManager

	CharSet() string

	DefaultTenantId() string

	TransOperationTimeout() int

	ServiceInvokeTimeout() int
}
