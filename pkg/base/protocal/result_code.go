package protocal

type ResultCode byte

const (

	// ResultCodeFailed failed
	ResultCodeFailed ResultCode = iota

	// ResultCodeSuccess success
	ResultCodeSuccess
)
