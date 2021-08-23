package event

type Manager struct {
	GlobalTransactionEventChannel chan GlobalTransactionEvent
}

var Bus Manager

func init() {
	Bus = Manager{GlobalTransactionEventChannel: make(chan GlobalTransactionEvent)}
}
