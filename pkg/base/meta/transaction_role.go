package meta

import (
	"fmt"
)

// TransactionRole
type TransactionRole byte

const (
	// tm
	TMRole TransactionRole = iota
	// rm
	RMRole
	// server
	ServerRole
)

// String string of transaction role
func (r TransactionRole) String() string {
	switch r {
	case TMRole:
		return "TMRole"
	case RMRole:
		return "RMRole"
	case ServerRole:
		return "ServerRole"
	default:
		return fmt.Sprintf("%d", r)
	}
}
