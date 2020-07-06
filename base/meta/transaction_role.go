package meta

import "fmt"

type TransactionRole byte

const (
	/**
	 * tm
	 */
	TMROLE TransactionRole = iota
	/**
	 * rm
	 */
	RMROLE
	/**
	 * server
	 */
	SERVERROLE
)

func (r TransactionRole) String() string {
	switch r {
	case TMROLE:
		return "TMROLE"
	case RMROLE:
		return "RMROLE"
	case SERVERROLE:
		return "SERVERROLE"
	default:
		return fmt.Sprintf("%d", r)
	}
}
