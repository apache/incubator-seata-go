package model

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
)

type Resource interface {
	GetResourceID() string

	GetBranchType() apis.BranchSession_BranchType
}
