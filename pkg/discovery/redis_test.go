package discovery

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisRegistryService_Lookup(t *testing.T) {
	db, _ := redismock.NewClientMock()
	type fields struct {
		config        *RedisConfig
		cli           *redis.Client
		rwLock        *sync.RWMutex
		vgroupMapping map[string]string
		groupList     map[string][]*ServiceInstance
		ctx           context.Context
	}
	type args struct {
		key string
	}
	redisConfig := &RedisConfig{
		Cluster: "default",
	}
	ctx := context.Background()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantR   []*ServiceInstance
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "default1",
			fields: fields{
				config:        redisConfig,
				cli:           db,
				rwLock:        &sync.RWMutex{},
				vgroupMapping: map[string]string{},
				groupList:     map[string][]*ServiceInstance{},
				ctx:           ctx,
			},
			args:  args{key: ""},
			wantR: make([]*ServiceInstance, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &RedisRegistryService{
				config:        tt.fields.config,
				cli:           tt.fields.cli,
				rwLock:        tt.fields.rwLock,
				vgroupMapping: tt.fields.vgroupMapping,
				groupList:     tt.fields.groupList,
				ctx:           tt.fields.ctx,
			}
			gotR, err := s.Lookup(tt.args.key)
			if !tt.wantErr(t, err, fmt.Sprintf("Lookup(%v)", tt.args.key)) {
				return
			}
			assert.Equalf(t, tt.wantR, gotR, "Lookup(%v)", tt.args.key)
		})
	}
}

func TestRedisCluster(t *testing.T) {
	input := "registry.redis.my_cluster_localhost:3306"
	regex := `registry\.redis\.(\w+)_`

	re := regexp.MustCompile(regex)
	match := re.FindStringSubmatch(input)

	if len(match) > 1 {
		cluster := match[1]
		fmt.Println("Cluster:", cluster)
	} else {
		fmt.Println("No match found.")
	}
}
