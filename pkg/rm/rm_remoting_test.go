package rm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRMRemotingInstance(t *testing.T) {
	tests := struct {
		name string
		want *RMRemoting
	}{"test1", &RMRemoting{}}

	t.Run(tests.name, func(t *testing.T) {
		assert.Equalf(t, tests.want, GetRMRemotingInstance(), "GetRMRemotingInstance()")
	})
}
