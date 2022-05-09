package protocol

// MessageFuture ...
type MessageFuture struct {
	ID       int32
	Err      error
	Response interface{}
	Done     chan bool
}

// NewMessageFuture ...
func NewMessageFuture(message RpcMessage) *MessageFuture {
	return &MessageFuture{
		ID:   message.ID,
		Done: make(chan bool),
	}
}
