package engine

type StateMachineConfig interface {
	GetStateLogStore() StateLogStore

	GetStateLangStore() StateLangStore
}
