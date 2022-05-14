package protocol

var MAGIC_CODE_BYTES = [2]byte{0xda, 0xda}

type RequestType byte

const (
	VERSION = 1

	// MaxFrameLength max frame length
	MaxFrameLength = 8 * 1024 * 1024

	// V1HeadLength v1 head length
	V1HeadLength = 16

	// MSGTypeRequestSync request message type
	MSGTypeRequestSync RequestType = 0

	// MSGTypeResponse response message type
	MSGTypeResponse RequestType = 1

	// MSGTypeRequestOneway request one way
	MSGTypeRequestOneway RequestType = 2

	// MSGTypeHeartbeatRequest heart beat request
	MSGTypeHeartbeatRequest RequestType = 3

	// MSGTypeHeartbeatResponse heart beat response
	MSGTypeHeartbeatResponse RequestType = 4
)
