package protocol

type MergedWarpMessage struct {
	Msgs   []MessageTypeAware
	MsgIds []int32
}

func (req MergedWarpMessage) GetTypeCode() MessageType {
	return MessageTypeSeataMerge
}

type MergeResultMessage struct {
	Msgs []MessageTypeAware
}

func (resp MergeResultMessage) GetTypeCode() MessageType {
	return MessageTypeSeataMergeResult
}
