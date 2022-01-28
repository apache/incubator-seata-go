package meta

import (
	"fmt"
)

// BranchTrype
type BranchType byte

const (
	// The At.
	// BranchType_AT Branch
	BranchTypeAT BranchType = iota

	//The BranchType_TCC.
	BranchTypeTCC

	// The BranchType_SAGA.
	BranchTypeSAGA
)

// String string of branch type
func (t BranchType) String() string {
	switch t {
	case BranchTypeAT:
		return "AT"
	case BranchTypeTCC:
		return "TCC"
	case BranchTypeSAGA:
		return "SAGA"
	default:
		return fmt.Sprintf("%d", t)
	}
}

// ValueOfBranchType value of branch type
func ValueOfBranchType(branchType string) BranchType {
	switch branchType {
	case "AT":
		return BranchTypeAT
	case "TCC":
		return BranchTypeTCC
	case "SAGA":
		return BranchTypeSAGA
	default:
		return 0
	}
}
