package engine

type StateMachineConfig interface {
	StateLogRepository() StateLogRepository

	StateMachineRepository() StateMachineRepository

	StateLogStore() StateLogStore

	StateLangStore() StateLangStore

	ExpressionFactoryManager() ExpressionFactoryManager

	ExpressionResolver() ExpressionResolver

	SeqGenerator() SeqGenerator

	StatusDecisionStrategy() StatusDecisionStrategy

	EventPublisher() EventPublisher

	AsyncEventPublisher() EventPublisher

	ServiceInvokerManager() ServiceInvokerManager

	ScriptInvokerManager() ScriptInvokerManager

	CharSet() string

	DefaultTenantId() string

	TransOperationTimeout() int

	ServiceInvokeTimeout() int
}
