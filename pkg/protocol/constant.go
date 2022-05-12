package protocol

var MAGIC_CODE_BYTES = [2]byte{0xda, 0xda}

type MessageType byte

const (
	VERSION = 1

	// MaxFrameLength max frame length
	MaxFrameLength = 8 * 1024 * 1024

	// V1HeadLength v1 head length
	V1HeadLength = 16

	// MSGTypeRequestSync request message type
	MSGTypeRequestSync MessageType = 0

	// MSGTypeResponse response message type
	MSGTypeResponse MessageType = 1

	// MSGTypeRequestOneway request one way
	MSGTypeRequestOneway MessageType = 2

	// MSGTypeHeartbeatRequest heart beat request
	MSGTypeHeartbeatRequest MessageType = 3

	// MSGTypeHeartbeatResponse heart beat response
	MSGTypeHeartbeatResponse MessageType = 4
)
