package message

type RpcMessage struct {
	ID         int32
	Type       GettyRequestType
	Codec      byte
	Compressor byte
	HeadMap    map[string]string
	Body       interface{}
}

type MessageFuture struct {
	ID       int32
	Err      error
	Response interface{}
	Done     chan bool
}

func NewMessageFuture(message RpcMessage) *MessageFuture {
	return &MessageFuture{
		ID:   message.ID,
		Done: make(chan bool),
	}
}

type HeartBeatMessage struct {
	Ping bool
}

var (
	HeartBeatMessagePing = HeartBeatMessage{true}
	HeartBeatMessagePong = HeartBeatMessage{false}
)

func (msg HeartBeatMessage) ToString() string {
	if msg.Ping {
		return "services ping"
	} else {
		return "services pong"
	}
}
