package model

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
)

// Resource used to manage transaction resource
type Resource interface {
	GetResourceID() string

	GetBranchType() apis.BranchSession_BranchType
}
