package proxy

type Propagation byte

const (
	/**
	 * The REQUIRED.
	 */
	REQUIRED Propagation = iota

	/**
	 * The REQUIRES_NEW.
	 */
	REQUIRES_NEW

	/**
	 * The NOT_SUPPORTED
	 */
	NOT_SUPPORTED

	/**
	 * The SUPPORTS
	 */
	SUPPORTS

	/**
	 * The NEVER
	 */
	NEVER

	/**
	 * The MANDATORY
	 */
	MANDATORY
)

type TransactionInfo struct {
	TimeOut int32
	Name string
	Propagation Propagation
}