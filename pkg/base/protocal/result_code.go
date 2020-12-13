package protocal

type ResultCode byte

const (

	/**
	 * ResultCodeFailed result code.
	 */
	// ResultCodeFailed
	ResultCodeFailed ResultCode = iota

	/**
	 * Success result code.
	 */
	// Success
	ResultCodeSuccess
)
