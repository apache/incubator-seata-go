package protocal

type AbstractResultMessage struct {
	ResultCode ResultCode
	Msg        string
}

type AbstractIdentifyRequest struct {
	Version string

	ApplicationID string

	TransactionServiceGroup string

	ExtraData []byte
}

type AbstractIdentifyResponse struct {
	AbstractResultMessage

	Version string

	ExtraData []byte

	Identified bool
}
