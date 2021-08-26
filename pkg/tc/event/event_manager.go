package event

type EventManager struct {
	GlobalTransactionEventChannel chan GlobalTransactionEvent
}

var EventBus EventManager

func init() {
	EventBus = EventManager{GlobalTransactionEventChannel: make(chan GlobalTransactionEvent)}
}
