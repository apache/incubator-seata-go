package protocal

type RpcMessage struct {
	ID          int32
	MessageType byte
	Codec       byte
	Compressor  byte
	HeadMap     map[string]string
	Body        interface{}
}
