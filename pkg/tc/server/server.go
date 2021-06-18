package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"
	gxnet "github.com/dubbogo/gost/net"
	"github.com/dubbogo/gost/sync"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common/extension"
	"github.com/transaction-wg/seata-golang/pkg/base/getty/readwriter"
	"github.com/transaction-wg/seata-golang/pkg/base/registry"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

type Server struct {
	conf       config.ServerConfig
	tcpServer  getty.Server
	rpcHandler *DefaultCoordinator
}

func NewServer() *Server {
	s := &Server{
		conf: config.GetServerConfig(),
	}
	coordinator := NewDefaultCoordinator(s.conf)
	s.rpcHandler = coordinator

	return s
}

func (s *Server) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
	)
	conf := s.conf

	if conf.GettyConfig.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}

	if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection\n", session.Stat(), session.Conn()))
	}

	tcpConn.SetNoDelay(conf.GettyConfig.GettySessionParam.TcpNoDelay)
	tcpConn.SetKeepAlive(conf.GettyConfig.GettySessionParam.TcpKeepAlive)
	if conf.GettyConfig.GettySessionParam.TcpKeepAlive {
		tcpConn.SetKeepAlivePeriod(conf.GettyConfig.GettySessionParam.KeepAlivePeriod)
	}
	tcpConn.SetReadBuffer(conf.GettyConfig.GettySessionParam.TcpRBufSize)
	tcpConn.SetWriteBuffer(conf.GettyConfig.GettySessionParam.TcpWBufSize)

	session.SetName(conf.GettyConfig.GettySessionParam.SessionName)
	session.SetMaxMsgLen(conf.GettyConfig.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(readwriter.RpcPkgHandler)
	session.SetEventListener(s.rpcHandler)
	session.SetReadTimeout(conf.GettyConfig.GettySessionParam.TcpReadTimeout)
	session.SetWriteTimeout(conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
	session.SetCronPeriod((int)(conf.GettyConfig.SessionTimeout.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettyConfig.GettySessionParam.WaitTimeout)
	log.Debugf("cmd accepts new session:%s\n", session.Stat())

	return nil
}

func (s *Server) Start(addr string) {
	var (
		tcpServer getty.Server
	)
	//直接使用addr绑定有ip，如果是127.0.0.1,则通过网卡正式ip不能访问
	addrs := strings.Split(addr, ":")
	tcpServer = getty.NewTCPServer(
		//getty.WithLocalAddress(addr),
		getty.WithLocalAddress(":"+addrs[1]),
		getty.WithServerTaskPool(gxsync.NewTaskPoolSimple(0)),
	)
	tcpServer.RunEventLoop(s.newSession)
	log.Debugf("s bind addr{%s} ok!", addr)
	s.tcpServer = tcpServer
	//向注册中心注册实例
	registryInstance()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-c
		log.Info("get a signal %s", sig.String())
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			s.Stop()
			time.Sleep(time.Second)
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}

func registryInstance() {
	registryConfig := config.GetRegistryConfig()
	reg, err := extension.GetRegistry(registryConfig.Mode)
	if err != nil {
		log.Error("Registry can not connect success, program is going to panic.Error message is %s", err.Error())
		panic(err.Error())
	}
	ip, _ := gxnet.GetLocalIP()
	conf := config.GetServerConfig()
	port, _ := strconv.Atoi(conf.Port)
	err = reg.Register(&registry.Address{
		IP:   ip,
		Port: uint64(port),
	})
	if err != nil {
		log.Error("Registry instance fail, %s", err.Error())
	}
}

func (s *Server) Stop() {
	s.tcpServer.Close()
	s.rpcHandler.Stop()
}
