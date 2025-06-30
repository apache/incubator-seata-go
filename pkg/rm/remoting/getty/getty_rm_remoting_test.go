package getty

import (
	"testing"

	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/rm"

	"github.com/stretchr/testify/assert"
)

func TestGetGettyRMRemotingInstance(t *testing.T) {
	rm.SetRMRemotingInstance(&GettyRMRemoting{})
	tests := []struct {
		name     string
		protocol string
		wantType interface{}
	}{
		{
			name:     "GettyRMRemoting",
			protocol: "seata",
			wantType: &GettyRMRemoting{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			config.InitTransportConfig(&config.TransportConfig{
				Protocol: tt.protocol,
			})

			got := rm.GetRMRemotingInstance()
			assert.NotNil(t, got)
			assert.IsType(t, tt.wantType, got)
		})
	}
}
