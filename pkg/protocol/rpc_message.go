package protocol

type RpcMessage struct {
	ID         int32
	Type       RequestType
	Codec      byte
	Compressor byte
	HeadMap    map[string]string
	Body       interface{}
}
