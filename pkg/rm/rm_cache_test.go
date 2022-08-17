package rm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/protocol/branch"
)

func TestGetRmCacheInstance(t *testing.T) {

	ctl := gomock.NewController(t)

	mockResourceManager := NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().GetBranchType().Return(branch.BranchTypeTCC)

	tests := struct {
		name string
		want *ResourceManagerCache
	}{"test1", &ResourceManagerCache{}}

	t.Run(tests.name, func(t *testing.T) {
		GetRmCacheInstance().RegisterResourceManager(mockResourceManager)
		actual := GetRmCacheInstance().GetResourceManager(branch.BranchTypeTCC)
		assert.Equalf(t, mockResourceManager, actual, "GetRmCacheInstance()")
	})

}
