package protocal

var MAGIC_CODE_BYTES = [2]byte{0xda, 0xda}

const (
	// version
	VERSION = 1

	// MaxFrameLength max frame length
	MaxFrameLength = 8 * 1024 * 1024

	// V1HeadLength v1 head length
	V1HeadLength = 16

	// MSGTypeRequest request message type
	MSGTypeRequest = 0

	// MSGTypeResponse response message type
	MSGTypeResponse = 1

	// MSGTypeRequestOneway request one way
	MSGTypeRequestOneway = 2

	// MSGTypeHeartbeatRequest heart beat request
	MSGTypeHeartbeatRequest = 3

	// MSGTypeHeartbeatResponse heart beat response
	MSGTypeHeartbeatResponse = 4
)
