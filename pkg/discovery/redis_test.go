package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newRedisRegisterService(t *testing.T) {
	type args struct {
		config      *ServiceConfig
		redisConfig *RedisConfig
	}
	redisConfig := &RedisConfig{
		Cluster:    "default",
		ServerAddr: "localhost:2379",
		Username:   "redis",
		Password:   "",
		DB:         0,
	}
	tests := []struct {
		name string
		args args
		want RegistryService
	}{
		{
			name: "default1",
			args: args{
				config:      nil,
				redisConfig: redisConfig,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, newRedisRegisterService(tt.args.config, tt.args.redisConfig), "newRedisRegisterService(%v, %v)", tt.args.config, tt.args.redisConfig)
		})
	}
}
