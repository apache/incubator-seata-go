package message

type AbstractResultMessage struct {
	ResultCode ResultCode
	Msg        string
}

type AbstractIdentifyRequest struct {
	Version                 string
	ApplicationId           string `json:"applicationId"`
	TransactionServiceGroup string
	ExtraData               []byte
}

type AbstractIdentifyResponse struct {
	AbstractResultMessage
	Version    string
	ExtraData  []byte
	Identified bool
}

type MessageTypeAware interface {
	GetTypeCode() MessageType
}

type MergedWarpMessage struct {
	Msgs   []MessageTypeAware
	MsgIds []int32
}

func (req MergedWarpMessage) GetTypeCode() MessageType {
	return MessageType_SeataMerge
}

type MergeResultMessage struct {
	Msgs []MessageTypeAware
}

func (resp MergeResultMessage) GetTypeCode() MessageType {
	return MessageType_SeataMergeResult
}

type ResultCode byte

const (
	ResultCodeFailed ResultCode = iota
	ResultCodeSuccess
)
