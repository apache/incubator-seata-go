package mock

import "go.etcd.io/etcd/client/v3"

type EtcdClient interface {
	clientv3.KV
	clientv3.Watcher
}
