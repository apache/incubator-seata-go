package protocol

type RpcMessage struct {
	ID          int32
	MessageType RequestType
	Codec       byte
	Compressor  byte
	HeadMap     map[string]string
	Body        interface{}
}
