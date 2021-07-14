package client

import (
	"fmt"
	"net"
)

import (
	getty "github.com/apache/dubbo-getty"
	gxsync "github.com/dubbogo/gost/sync"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/getty/readwriter"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	getty2 "github.com/transaction-wg/seata-golang/pkg/client/rpc_client"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

type RpcClient struct {
	conf         *config.ClientConfig
	gettyClients []getty.Client
	rpcHandler   *getty2.RpcRemoteClient
}

func NewRpcClient() *RpcClient {
	rpcClient := &RpcClient{
		conf:         config.GetClientConfig(),
		gettyClients: make([]getty.Client, 0),
		rpcHandler:   getty2.InitRpcRemoteClient(),
	}
	rpcClient.init()
	return rpcClient
}

func (c *RpcClient) init() {
	addressList := getAvailServerList(c.conf)
	if len(addressList) == 0 {
		log.Warn("no have valid seata server list")
	}
	for _, address := range addressList {
		gettyClient := getty.NewTCPClient(
			getty.WithServerAddress(address),
			getty.WithConnectionNumber((int)(c.conf.GettyConfig.ConnectionNum)),
			getty.WithReconnectInterval(c.conf.GettyConfig.ReconnectInterval),
			getty.WithClientTaskPool(gxsync.NewTaskPoolSimple(0)),
		)
		go gettyClient.RunEventLoop(c.newSession)
		c.gettyClients = append(c.gettyClients, gettyClient)
	}
}

func getAvailServerList(config *config.ClientConfig) []string {
	reg, err := extension.GetRegistry(config.RegistryConfig.Mode)
	if err != nil {
		logger.Errorf("Registry can not connect success, program is going to panic.Error message is %s", err.Error())
		panic(err.Error())
	}
	addrs, err := reg.Lookup()
	if err != nil {
		logger.Errorf("no hava valid server list", err.Error())
		return nil
	}
	return addrs
}

func (c *RpcClient) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
	)

	if c.conf.GettyConfig.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}

	if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection\n", session.Stat(), session.Conn()))
	}

	tcpConn.SetNoDelay(c.conf.GettyConfig.GettySessionParam.TCPNoDelay)
	tcpConn.SetKeepAlive(c.conf.GettyConfig.GettySessionParam.TCPKeepAlive)
	if c.conf.GettyConfig.GettySessionParam.TCPKeepAlive {
		tcpConn.SetKeepAlivePeriod(c.conf.GettyConfig.GettySessionParam.KeepAlivePeriod)
	}
	tcpConn.SetReadBuffer(c.conf.GettyConfig.GettySessionParam.TCPRBufSize)
	tcpConn.SetWriteBuffer(c.conf.GettyConfig.GettySessionParam.TCPWBufSize)

	session.SetName(c.conf.GettyConfig.GettySessionParam.SessionName)
	session.SetMaxMsgLen(c.conf.GettyConfig.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(readwriter.RpcPkgHandler)
	session.SetEventListener(c.rpcHandler)
	session.SetReadTimeout(c.conf.GettyConfig.GettySessionParam.TCPReadTimeout)
	session.SetWriteTimeout(c.conf.GettyConfig.GettySessionParam.TCPWriteTimeout)
	session.SetCronPeriod((int)(c.conf.GettyConfig.HeartbeatPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(c.conf.GettyConfig.GettySessionParam.WaitTimeout)
	log.Debugf("rpc_client new session:%s\n", session.Stat())

	return nil
}
