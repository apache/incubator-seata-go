package discovery

import (
	"context"
	etcd3 "go.etcd.io/etcd/client/v3"
	"reflect"
	"testing"
	"time"
)

func TestEtcdRegistryService_Lookup(t *testing.T) {
	type fields struct {
		serviceConfig  *ServiceConfig
		registryConfig *RegistryConfig
	}
	type args struct {
		key        string
		etcdPrefix string
	}
	type etcdPut struct {
		putMap map[string]string
	}
	type etcdDel struct {
		delList []string
	}
	type operation struct {
		put etcdPut
		del etcdDel
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		ops        operation
		want       []*ServiceInstance
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "normal single endpoint.",
			args: args{
				key: "default_tx_group",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
				},
				// change to your own configuration
				registryConfig: &RegistryConfig{
					Type: ETCD,
					Etcd3: Etcd3Config{
						ServerAddr: "127.0.0.1:23799",
						Cluster:    "default",
					},
				},
			},
			want: []*ServiceInstance{
				{
					Addr: "172.20.0.4",
					Port: 8091,
				},
			},
			wantErr: false,
		},
		{
			name: "put a new endpoint.",
			args: args{
				key:        "default_tx_group",
				etcdPrefix: "registry-seata-",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
				},
				registryConfig: &RegistryConfig{
					Type: ETCD,
					Etcd3: Etcd3Config{
						ServerAddr: "127.0.0.1:23799",
						Cluster:    "default",
					},
				},
			},
			want: []*ServiceInstance{
				{
					Addr: "172.20.0.4",
					Port: 8091,
				},
				{
					Addr: "172.0.0.1",
					Port: 8091,
				},
			},
			ops: operation{
				put: etcdPut{
					putMap: map[string]string{
						"default-172.0.0.1:8091": "172.0.0.1:8091",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "delete a outdated endpoint.",
			args: args{
				key:        "default_tx_group",
				etcdPrefix: "registry-seata-",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
				},
				registryConfig: &RegistryConfig{
					Type: ETCD,
					Etcd3: Etcd3Config{
						ServerAddr: "127.0.0.1:23799",
						Cluster:    "default",
					},
				},
			},
			want: []*ServiceInstance{
				{
					Addr: "172.20.0.4",
					Port: 8091,
				},
			},
			ops: operation{
				del: etcdDel{
					delList: []string{"default-172.0.0.1:8091"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitRegistry(tt.fields.serviceConfig, tt.fields.registryConfig)
			s := GetRegistry()
			// wait a second to set up watch
			time.Sleep(1 * time.Second)
			got, err := s.Lookup(tt.args.key)
			if err != nil {
				t.Errorf("Got some err when run Lookup(). error = %e", err)
			}
			t.Log("Before etcd operation")

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Log("Lookup() got ")
				for _, instance := range got {
					t.Logf("%v\n", instance)
				}
				t.Log("want got ")
				for _, want := range tt.want {
					t.Logf("%v\n", want)
				}
			}

			cfg := etcd3.Config{
				Endpoints: []string{tt.fields.registryConfig.Etcd3.ServerAddr},
			}
			cli, err := etcd3.New(cfg)
			if err != nil {
				t.Errorf("Got some err when create etcd-cli. error = %e", err)
			}
			put := tt.ops.put
			timeout := 5 * time.Second
			if len(put.putMap) != 0 {
				for key, value := range put.putMap {
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					cli.Put(ctx, tt.args.etcdPrefix+key, value)
				}
			}

			del := tt.ops.del
			if len(del.delList) != 0 {
				for _, key := range del.delList {
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					cli.Delete(ctx, tt.args.etcdPrefix+key)
				}
			}
			t.Log("After etcd operation")
			got, err = s.Lookup(tt.args.key)
			if err != nil {
				t.Errorf("Got some err when run Lookup(). error = %e", err)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Lookup() got = %v, want = %v", got, tt.want)
			}
		})
	}
}
