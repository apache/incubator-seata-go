package core

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"sync"
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
