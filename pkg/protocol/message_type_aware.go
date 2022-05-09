package protocol

type MessageTypeAware interface {
	GetTypeCode() int16
}
