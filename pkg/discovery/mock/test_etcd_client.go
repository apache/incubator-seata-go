package mock

import clientv3 "go.etcd.io/etcd/client/v3"

type EtcdClient interface {
	clientv3.KV
	clientv3.Watcher
}
