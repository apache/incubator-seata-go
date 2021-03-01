package protocal

type MergedWarpMessage struct {
	Msgs   []MessageTypeAware
	MsgIDs []int32
}

func (req MergedWarpMessage) GetTypeCode() int16 {
	return TypeSeataMerge
}

type MergeResultMessage struct {
	Msgs []MessageTypeAware
}

func (resp MergeResultMessage) GetTypeCode() int16 {
	return TypeSeataMergeResult
}
