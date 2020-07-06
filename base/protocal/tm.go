package protocal

type RegisterTMRequest struct {
	AbstractIdentifyRequest
}

func (req RegisterTMRequest) GetTypeCode() int16 {
	return TypeRegClt
}

type RegisterTMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterTMResponse) GetTypeCode() int16 {
	return TypeRegCltResult
}
