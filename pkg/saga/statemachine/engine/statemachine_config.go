package engine

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/events"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/status_decision"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/store"
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
