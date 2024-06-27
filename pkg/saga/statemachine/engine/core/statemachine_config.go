package core

import (
	"sync"

	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/expr"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/invoker"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/sequence"
)

type StateMachineConfig interface {
	StateLogRepository() StateLogRepository

	StateMachineRepository() StateMachineRepository

	StateLogStore() StateLogStore

	StateLangStore() StateLangStore

	ExpressionFactoryManager() expr.ExpressionFactoryManager

	ExpressionResolver() expr.ExpressionResolver

	SeqGenerator() sequence.SeqGenerator

	StatusDecisionStrategy() StatusDecisionStrategy

	EventPublisher() EventPublisher

	AsyncEventPublisher() EventPublisher

	ServiceInvokerManager() invoker.ServiceInvokerManager

	ScriptInvokerManager() invoker.ScriptInvokerManager

	CharSet() string

	DefaultTenantId() string

	TransOperationTimeout() int

	ServiceInvokeTimeout() int

	ComponentLock() *sync.Mutex
}
