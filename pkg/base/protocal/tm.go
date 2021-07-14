package protocal

// RegisterTMRequest
type RegisterTMRequest struct {
	AbstractIdentifyRequest
}

// GetTypeCode
func (req RegisterTMRequest) GetTypeCode() int16 {
	return TypeRegClt
}

// RegisterTMResponse
type RegisterTMResponse struct {
	AbstractIdentifyResponse
}

// GetTypeCode
func (resp RegisterTMResponse) GetTypeCode() int16 {
	return TypeRegCltResult
}
