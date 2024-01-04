package engine

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/events"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/status_decision"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/store"
)

const (
	DefaultTransOperTimeout     = 60000 * 30
	DefaultServiceInvokeTimeout = 60000 * 5
)

type DefaultStateMachineConfig struct {
	// Configuration
	transOperationTimeout int
	serviceInvokeTimeout  int
	charset               string
	defaultTenantId       string

	// Components

	// Store related components
	stateLogRepository     store.StateLogRepository
	stateLogStore          store.StateLogStore
	stateLangStore         store.StateLangStore
	stateMachineRepository store.StateMachineRepository

	// Expression related components
	expressionFactoryManager expr.ExpressionFactoryManager
	expressionResolver       expr.ExpressionResolver

	// Invoker related components
	serviceInvokerManager invoker.ServiceInvokerManager
	scriptInvokerManager  invoker.ScriptInvokerManager

	// Other components
	statusDecisionStrategy status_decision.StatusDecisionStrategy
	seqGenerator           sequence.SeqGenerator
}

func (c *DefaultStateMachineConfig) StateLogRepository() store.StateLogRepository {
	return c.stateLogRepository
}

func (c *DefaultStateMachineConfig) StateMachineRepository() store.StateMachineRepository {
	return c.stateMachineRepository
}

func (c *DefaultStateMachineConfig) StateLogStore() store.StateLogStore {
	return c.stateLogStore
}

func (c *DefaultStateMachineConfig) StateLangStore() store.StateLangStore {
	return c.stateLangStore
}

func (c *DefaultStateMachineConfig) ExpressionFactoryManager() expr.ExpressionFactoryManager {
	return c.expressionFactoryManager
}

func (c *DefaultStateMachineConfig) ExpressionResolver() expr.ExpressionResolver {
	return c.expressionResolver
}

func (c *DefaultStateMachineConfig) SeqGenerator() sequence.SeqGenerator {
	return c.seqGenerator
}

func (c *DefaultStateMachineConfig) StatusDecisionStrategy() status_decision.StatusDecisionStrategy {
	return c.statusDecisionStrategy
}

func (c *DefaultStateMachineConfig) EventPublisher() events.EventPublisher {
	//TODO implement me
	panic("implement me")
}

func (c *DefaultStateMachineConfig) AsyncEventPublisher() events.EventPublisher {
	//TODO implement me
	panic("implement me")
}

func (c *DefaultStateMachineConfig) ServiceInvokerManager() invoker.ServiceInvokerManager {
	return c.serviceInvokerManager
}

func (c *DefaultStateMachineConfig) ScriptInvokerManager() invoker.ScriptInvokerManager {
	return c.scriptInvokerManager
}

func (c *DefaultStateMachineConfig) CharSet() string {
	return c.charset
}

func (c *DefaultStateMachineConfig) SetCharSet(charset string) {
	c.charset = charset
}

func (c *DefaultStateMachineConfig) DefaultTenantId() string {
	return c.defaultTenantId
}

func (c *DefaultStateMachineConfig) TransOperationTimeout() int {
	return c.transOperationTimeout
}

func (c *DefaultStateMachineConfig) ServiceInvokeTimeout() int {
	return c.serviceInvokeTimeout
}

func NewDefaultStateMachineConfig() *DefaultStateMachineConfig {
	c := &DefaultStateMachineConfig{
		transOperationTimeout: DefaultTransOperTimeout,
		serviceInvokeTimeout:  DefaultServiceInvokeTimeout,
		charset:               "UTF-8",
		defaultTenantId:       "000001",
	}
	return c
}
