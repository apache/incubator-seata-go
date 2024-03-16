package discovery

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisRegistryService_Lookup(t *testing.T) {
	cli, _ := redismock.NewClientMock()

	type fields struct {
		config    *RedisConfig
		cli       *redis.Client
		serverMap *sync.Map
		ctx       context.Context
	}
	config := &RedisConfig{
		Cluster:    "default",
		ServerAddr: "localhost:6379",
		Username:   "",
		Password:   "123456",
		DB:         0,
	}
	type args struct {
		key string
	}
	ctx := context.Background()
	_, err := cli.Ping(ctx).Result()
	if err != nil {
		fmt.Println("ping error", err)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantR   []*ServiceInstance
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "default",
			fields: fields{
				config:    config,
				cli:       cli,
				serverMap: &sync.Map{},
				ctx:       ctx,
			},
			args:    args{key: "registry.redis.apple"},
			wantR:   nil,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgRedis := &RedisRegistryService{
				config:    tt.fields.config,
				cli:       tt.fields.cli,
				serverMap: tt.fields.serverMap,
				ctx:       tt.fields.ctx,
			}
			s := newRedisRegisterService(nil, cfgRedis.config)
			time.Sleep(1 * time.Second)
			gotR, err := s.Lookup(tt.args.key)
			if err != nil {
				return
			}
			for _, v := range gotR {
				fmt.Println(v)
			}
		})
	}
}
