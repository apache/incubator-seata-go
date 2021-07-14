package protocal

// AbstractResultMessage
type AbstractResultMessage struct {
	ResultCode ResultCode
	Msg        string
}

// AbstractIdentifyRequest
type AbstractIdentifyRequest struct {
	Version string

	ApplicationID string

	TransactionServiceGroup string

	ExtraData []byte
}

// AbstractIdentifyResponse
type AbstractIdentifyResponse struct {
	AbstractResultMessage

	Version string

	ExtraData []byte

	Identified bool
}
