package protocal

type RegisterRMRequest struct {
	AbstractIdentifyRequest
	ResourceIDs string
}

func (req RegisterRMRequest) GetTypeCode() int16 {
	return TypeRegRm
}

type RegisterRMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterRMResponse) GetTypeCode() int16 {
	return TypeRegRmResult
}
