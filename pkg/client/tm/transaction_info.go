package tm

import "fmt"

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

func (t Propagation) String() string {
	switch t {
	case REQUIRED:
		return "REQUIRED"
	case REQUIRES_NEW:
		return "REQUIRES_NEW"
	case NOT_SUPPORTED:
		return "NOT_SUPPORTED"
	case SUPPORTS:
		return "SUPPORTS"
	case NEVER:
		return "NEVER"
	case MANDATORY:
		return "MANDATORY"
	default:
		return fmt.Sprintf("%d", t)
	}
}

type TransactionInfo struct {
	TimeOut     int32
	Name        string
	Propagation Propagation
}
