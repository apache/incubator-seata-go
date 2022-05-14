package protocol

type MessageTypeAware interface {
	GetTypeCode() MessageType
}
