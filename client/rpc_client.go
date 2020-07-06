package client

import (
	"fmt"
	"net"
	"strings"
)

import (
	"github.com/dubbogo/getty"
	gxsync "github.com/dubbogo/gost/sync"
)

import (
	"github.com/dk-lockdown/seata-golang/base/getty/readwriter"
	"github.com/dk-lockdown/seata-golang/client/config"
	getty2 "github.com/dk-lockdown/seata-golang/client/getty"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
)

var clientGrpool *gxsync.TaskPool

type RpcClient struct {
	conf         config.ClientConfig
	gettyClients []getty.Client
	rpcHandler   *getty2.RpcRemoteClient
}

func setClientGrpool() {
	clientConf := config.GetClientConfig()
	if clientConf.GettyConfig.GrPoolSize > 1 {
		clientGrpool = gxsync.NewTaskPool(gxsync.WithTaskPoolTaskPoolSize(clientConf.GettyConfig.GrPoolSize), gxsync.WithTaskPoolTaskQueueLength(clientConf.GettyConfig.QueueLen),
			gxsync.WithTaskPoolTaskQueueNumber(clientConf.GettyConfig.QueueNumber))
	}
}

func NewRpcClient() *RpcClient {
	setClientGrpool()
	rpcClient := &RpcClient{
		conf:         config.GetClientConfig(),
		gettyClients: make([]getty.Client, 0),
		rpcHandler:   getty2.InitRpcRemoteClient(),
	}
	rpcClient.init()
	return rpcClient
}

func (c *RpcClient) init() {
	addressList := strings.Split(c.conf.TransactionServiceGroup, ",")
	for _, address := range addressList {
		gettyClient := getty.NewTCPClient(
			getty.WithServerAddress(address),
			getty.WithConnectionNumber((int)(c.conf.GettyConfig.ConnectionNum)),
			getty.WithReconnectInterval(c.conf.GettyConfig.ReconnectInterval),
		)
		go gettyClient.RunEventLoop(c.newSession)
		c.gettyClients = append(c.gettyClients, gettyClient)
	}
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

	tcpConn.SetNoDelay(c.conf.GettyConfig.GettySessionParam.TcpNoDelay)
	tcpConn.SetKeepAlive(c.conf.GettyConfig.GettySessionParam.TcpKeepAlive)
	if c.conf.GettyConfig.GettySessionParam.TcpKeepAlive {
		tcpConn.SetKeepAlivePeriod(c.conf.GettyConfig.GettySessionParam.KeepAlivePeriod)
	}
	tcpConn.SetReadBuffer(c.conf.GettyConfig.GettySessionParam.TcpRBufSize)
	tcpConn.SetWriteBuffer(c.conf.GettyConfig.GettySessionParam.TcpWBufSize)

	session.SetName(c.conf.GettyConfig.GettySessionParam.SessionName)
	session.SetMaxMsgLen(c.conf.GettyConfig.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(readwriter.RpcPkgHandler)
	session.SetEventListener(c.rpcHandler)
	session.SetWQLen(c.conf.GettyConfig.GettySessionParam.PkgWQSize)
	session.SetReadTimeout(c.conf.GettyConfig.GettySessionParam.TcpReadTimeout)
	session.SetWriteTimeout(c.conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
	session.SetCronPeriod((int)(c.conf.GettyConfig.HeartbeatPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(c.conf.GettyConfig.GettySessionParam.WaitTimeout)
	logging.Logger.Debugf("client new session:%s\n", session.Stat())

	session.SetTaskPool(clientGrpool)

	return nil
}
