package protocol

var MAGIC_CODE_BYTES = [2]byte{0xda, 0xda}

type RequestType byte

const (
	VERSION = 1

	// MaxFrameLength max frame length
	MaxFrameLength = 8 * 1024 * 1024

	// V1HeadLength v1 head length
	V1HeadLength = 16

	// Request message type
	MSGTypeRequestSync RequestType = 0

	// Response message type
	MSGTypeResponse RequestType = 1

	// Request which no need response
	MSGTypeRequestOneway RequestType = 2

	// Heartbeat Request
	MSGTypeHeartbeatRequest RequestType = 3

	// Heartbeat Response
	MSGTypeHeartbeatResponse RequestType = 4
)
