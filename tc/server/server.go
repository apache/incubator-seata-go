package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

import (
	"github.com/dubbogo/getty"
	"github.com/dubbogo/gost/sync"
)

import (
	"github.com/dk-lockdown/seata-golang/base/getty/readwriter"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/config"
)

var (
	srvGrpool *gxsync.TaskPool
)

func SetServerGrpool() {
	srvConf := config.GetServerConfig()
	if srvConf.GettyConfig.GrPoolSize > 1 {
		srvGrpool = gxsync.NewTaskPool(gxsync.WithTaskPoolTaskPoolSize(srvConf.GettyConfig.GrPoolSize),
			gxsync.WithTaskPoolTaskQueueLength(srvConf.GettyConfig.QueueLen),
			gxsync.WithTaskPoolTaskQueueNumber(srvConf.GettyConfig.QueueNumber))
	}
}

type Server struct {
	conf       config.ServerConfig
	tcpServer  getty.Server
	rpcHandler *DefaultCoordinator
}

func NewServer() *Server {
	SetServerGrpool()
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
	session.SetWQLen(conf.GettyConfig.GettySessionParam.PkgWQSize)
	session.SetReadTimeout(conf.GettyConfig.GettySessionParam.TcpReadTimeout)
	session.SetWriteTimeout(conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
	session.SetCronPeriod((int)(conf.GettyConfig.SessionTimeout.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettyConfig.GettySessionParam.WaitTimeout)
	logging.Logger.Debugf("app accepts new session:%s\n", session.Stat())

	session.SetTaskPool(srvGrpool)

	return nil
}

func (s *Server) Start(addr string) {
	var (
		tcpServer getty.Server
	)

	tcpServer = getty.NewTCPServer(
		getty.WithLocalAddress(addr),
	)
	tcpServer.RunEventLoop(s.newSession)
	logging.Logger.Debugf("s bind addr{%s} ok!", addr)
	s.tcpServer = tcpServer

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-c
		logging.Logger.Info("get a signal %s", sig.String())
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

func (s *Server) Stop() {
	s.tcpServer.Close()
	s.rpcHandler.Stop()
}
