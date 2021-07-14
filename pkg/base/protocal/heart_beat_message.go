package protocal

// HeartBeatMessage
type HeartBeatMessage struct {
	Ping bool
}

// HeartBeatMessagePing
var HeartBeatMessagePing = HeartBeatMessage{true}

// HeartBeatMessagePong
var HeartBeatMessagePong = HeartBeatMessage{false}

// ToString
func (msg HeartBeatMessage) ToString() string {
	if msg.Ping {
		return "services ping"
	}
	return "services pong"
}
