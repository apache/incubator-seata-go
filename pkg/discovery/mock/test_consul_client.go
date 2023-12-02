package mock

import (
	"github.com/hashicorp/consul/sdk/testutil"
)

type ConsulClient interface {
	testutil.TestKVResponse
}
