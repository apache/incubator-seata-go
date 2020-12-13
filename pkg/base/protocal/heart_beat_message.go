package protocal

type HeartBeatMessage struct {
	Ping bool
}

var HeartBeatMessagePing = HeartBeatMessage{true}
var HeartBeatMessagePong = HeartBeatMessage{false}

func (msg HeartBeatMessage) ToString() string {
	if msg.Ping {
		return "services ping"
	} else {
		return "services pong"
	}
}
