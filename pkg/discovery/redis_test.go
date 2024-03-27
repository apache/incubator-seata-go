package discovery

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRedisRegistryService_Lookup(t *testing.T) {
	// db, mockClient := redismock.NewClientMock()
	serviceConfig := &ServiceConfig{
		VgroupMapping: map[string]string{
			"default_tx_group": "default",
		},
	}
	type args struct {
		key string
	}
	redisConfig := &RedisConfig{
		Cluster:    "default",
		ServerAddr: "localhost:6379",
		Username:   "",
		Password:   "123456",
		DB:         0,
	}
	// ctrl := gomock.NewController(t)
	// mockRedisClient := mock.NewMockRedisClient(ctrl)
	tests := []struct {
		name  string
		args  args
		wantR []*ServiceInstance
	}{
		{
			name: "default",
			args: args{key: "registry.redis.default_localhost:8888"},
			wantR: []*ServiceInstance{
				{
					Addr: "localhost",
					Port: 8888,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newRedisRegisterService(serviceConfig, redisConfig)
			// mockClient.ExpectSet("registry.redis.default_localhost:8888", "localhost:8888", -1)
			// wait 2 second for update all service
			time.Sleep(5 * time.Second)
			// result := mockClient.ExpectGet("registry.redis.default_localhost:8888")
			// fmt.Println("result", result)
			serviceInstances, err := s.Lookup("default_tx_group")
			if err != nil {
				t.Errorf("error happen when look up . err = %s", err)
				return
			}
			t.Logf("name:%s,key:%s,server length:%d", tt.name, tt.args.key, len(serviceInstances))
			for i := range serviceInstances {
				t.Log(serviceInstances[i].Addr)
				t.Log(serviceInstances[i].Port)
			}
			assert.True(t, reflect.DeepEqual(serviceInstances, tt.wantR))

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
