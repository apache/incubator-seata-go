package discovery

import (
	"github.com/golang/mock/gomock"
	"github.com/seata/seata-go/pkg/discovery/mock"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"reflect"
	"testing"
	"time"
)

func TestEtcd3RegistryService_Lookup(t *testing.T) {

	tests := []struct {
		name          string
		getResp       *clientv3.GetResponse
		watchResponse *clientv3.WatchResponse
		want          []*ServiceInstance
	}{
		{
			name: "normal",
			getResp: &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("registry-seata-default-172.0.0.1:8091"),
						Value: []byte("172.0.0.1:8091"),
					},
				},
			},
			watchResponse: nil,
			want: []*ServiceInstance{
				{
					Addr: "172.0.0.1",
					Port: 8091,
				},
			},
		},
		{
			name:    "use watch update ServiceInstances",
			getResp: nil,
			watchResponse: &clientv3.WatchResponse{
				Events: []*clientv3.Event{
					{
						Type: clientv3.EventTypePut,
						Kv: &mvccpb.KeyValue{
							Key:   []byte("registry-seata-default-172.0.0.1:8091"),
							Value: []byte("172.0.0.1:8091"),
						},
					},
				},
			},
			want: []*ServiceInstance{
				{
					Addr: "172.0.0.1",
					Port: 8091,
				},
			},
		},
	}

	for _, tt := range tests {
		ctrl := gomock.NewController(t)
		mockEtcdClient := mock.NewMockEtcdClient(ctrl)
		etcdRegistryService := &EtcdRegistryService{
			client: &clientv3.Client{
				KV:      mockEtcdClient,
				Watcher: mockEtcdClient,
			},
			vgroupMapping: map[string]string{
				"default_tx_group": "default",
			},
			grouplist: make(map[string][]*ServiceInstance, 0),
			stopCh:    make(chan struct{}),
		}

		mockEtcdClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.getResp, nil)
		ch := make(chan clientv3.WatchResponse)
		mockEtcdClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any()).Return(ch)

		go func() {
			etcdRegistryService.watch("registry-seata")
		}()
		// wait a second for watch
		time.Sleep(1 * time.Second)

		if tt.watchResponse != nil {
			go func() {
				ch <- *tt.watchResponse
			}()
		}

		// wait one more second for update
		time.Sleep(1 * time.Second)
		serviceInstances, err := etcdRegistryService.Lookup("default_tx_group")
		if err != nil {
			t.Errorf("error happen when look up . err = %e", err)
		}
		t.Logf(tt.name)
		for i := range serviceInstances {
			t.Log(serviceInstances[i].Addr)
			t.Log(serviceInstances[i].Port)
		}
		assert.True(t, reflect.DeepEqual(serviceInstances, tt.want))
	}
}
