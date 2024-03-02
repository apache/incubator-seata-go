package mock

import "go.etcd.io/etcd/clientv3"

type EtcdClient interface {
	clientv3.KV
	clientv3.Watcher
}
