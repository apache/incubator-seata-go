package etcdv3

// TODO: Import Standard
import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"time"
)

import (
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/constant"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
)

func IsAddressValid(addr registry.Address) bool {
	return len(addr.IP) != 0 || addr.Port != 0
}

func BuildRegistryKey(clusterName string, addr *registry.Address) string {
	return constant.ETCDV3_REGISTRY_PREFIX + fmt.Sprintf("%s-%s", clusterName, addrToStr(addr))
}

func BuildRegistryValue(addr *registry.Address) string {
	return addrToStr(addr)
}

func addrToStr(addr *registry.Address) string {
	return addr.IP + ":" + strconv.FormatUint(addr.Port, 10)
}

func ToEtcdConfig(cfg config.EtcdConfig, ctx context.Context) (clientv3.Config, error) {
	// Endpoints eg: "127.0.0.1:11451,127.0.0.1:11452"
	endpoints := strings.Split(cfg.Endpoints, ",")

	var tlsConfig *tls.Config
	if cfg.TLSConfig != nil {
		tcpInfo := transport.TLSInfo{
			CertFile:      cfg.TLSConfig.CertFile,
			KeyFile:       cfg.TLSConfig.KeyFile,
			TrustedCAFile: cfg.TLSConfig.TrustedCAFile,
		}

		var err error
		tlsConfig, err = tcpInfo.ClientConfig()
		if err != nil {
			return clientv3.Config{}, err
		}
	}

	autoSyncInterval, err := time.ParseDuration(cfg.AutoSyncInterval)
	if err != nil {
		return clientv3.Config{}, err
	}

	dialTimeout, err := time.ParseDuration(cfg.DialTimeout)
	if err != nil {
		return clientv3.Config{}, err
	}

	keepAliveTime, err := time.ParseDuration(cfg.DialKeepAliveTime)
	if err != nil {
		return clientv3.Config{}, err
	}

	keepAliveTimeOut, err := time.ParseDuration(cfg.DialKeepAliveTimeout)
	if err != nil {
		return clientv3.Config{}, err
	}

	return clientv3.Config{
		Endpoints:            endpoints,
		AutoSyncInterval:     autoSyncInterval,
		DialTimeout:          dialTimeout,
		DialKeepAliveTime:    keepAliveTime,
		DialKeepAliveTimeout: keepAliveTimeOut,
		MaxCallSendMsgSize:   cfg.MaxCallSendMsgSize,
		MaxCallRecvMsgSize:   cfg.MaxCallRecvMsgSize,
		Username:             cfg.Username,
		Password:             cfg.Password,
		RejectOldCluster:     false,
		DialOptions:          []grpc.DialOption{grpc.WithBlock()}, // Disable Async
		PermitWithoutStream:  true,
		TLS:                  tlsConfig,
		Context:              ctx,
	}, nil
}
