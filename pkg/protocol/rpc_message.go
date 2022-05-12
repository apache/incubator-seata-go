package protocol

type RpcMessage struct {
	ID          int32
	MessageType MessageType
	Codec       byte
	Compressor  byte
	HeadMap     map[string]string
	Body        interface{}
}
