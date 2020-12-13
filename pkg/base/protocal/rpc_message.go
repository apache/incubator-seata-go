package protocal

type RpcMessage struct {
	Id          int32
	MessageType byte
	Codec       byte
	Compressor  byte
	HeadMap     map[string]string
	Body        interface{}
}
