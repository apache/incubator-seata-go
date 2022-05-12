package model

// Resource that can be managed by Resource Manager and involved into global transaction
type Resource interface {
	GetResourceGroupId() string
	GetResourceId() string
	GetBranchType() BranchType
}
