package protocol

type RegisterTMRequest struct {
	AbstractIdentifyRequest
}

func (req RegisterTMRequest) GetTypeCode() MessageType {
	return MessageTypeRegClt
}

type RegisterTMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterTMResponse) GetTypeCode() MessageType {
	return MessageTypeRegCltResult
}
