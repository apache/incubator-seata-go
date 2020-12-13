package protocal

var MAGIC_CODE_BYTES = [2]byte{0xda, 0xda}

const (
	VERSION                    = 1
	MAX_FRAME_LENGTH           = 8 * 1024 * 1024
	V1_HEAD_LENGTH             = 16
	MSGTYPE_RESQUEST           = 0
	MSGTYPE_RESPONSE           = 1
	MSGTYPE_RESQUEST_ONEWAY    = 2
	MSGTYPE_HEARTBEAT_REQUEST  = 3
	MSGTYPE_HEARTBEAT_RESPONSE = 4
)
