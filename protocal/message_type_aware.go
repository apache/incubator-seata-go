package protocal

type MessageTypeAware interface {
	GetTypeCode() int16
}
