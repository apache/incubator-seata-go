package protocol

type RegisterRMRequest struct {
	AbstractIdentifyRequest
	ResourceIds string
}

func (req RegisterRMRequest) GetTypeCode() MessageType {
	return MessageTypeRegRm
}

type RegisterRMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterRMResponse) GetTypeCode() MessageType {
	return MessageTypeRegRmResult
}
