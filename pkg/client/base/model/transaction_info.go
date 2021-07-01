package model

import "fmt"

// Propagation transaction isolation level
type Propagation byte

const (
	Required Propagation = iota

	RequiresNew

	NotSupported

	Supports

	Never

	Mandatory
)

// String
func (t Propagation) String() string {
	switch t {
	case Required:
		return "Required"
	case RequiresNew:
		return "REQUIRES_NEW"
	case NotSupported:
		return "NOT_SUPPORTED"
	case Supports:
		return "Supports"
	case Never:
		return "Never"
	case Mandatory:
		return "Mandatory"
	default:
		return fmt.Sprintf("%d", t)
	}
}

// TransactionInfo used to configure global transaction parameters
type TransactionInfo struct {
	TimeOut     int32
	Name        string
	Propagation Propagation
}
