package rpc11

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

import (
	getty "github.com/apache/dubbo-getty"

	gxsync "github.com/dubbogo/gost/sync"
)

import (
	"github.com/seata/seata-go/pkg/config"
	log "github.com/seata/seata-go/pkg/util/log"
)

var (
	rpcClient             *RpcClient
	MAX_CHECK_ALIVE_RETRY = 600
	CHECK_ALIVE_INTERNAL  = 100
)

func init() {
	rpcClient = newRpcClient()
}

type RpcClient struct {
	lock         sync.RWMutex
	conf         *config.ClientConfig
	gettyClients []getty.Client
	listener     getty.EventListener
	// serverAddress -> rpc_client.Session -> bool
	serverSessionsMap sync.Map
	sessionSize       int32
}

func newRpcClient() *RpcClient {
	rpcClient := &RpcClient{
		conf:         config.GetClientConfig(),
		gettyClients: make([]getty.Client, 0),
		listener:     NewClientEventHandler(),
	}
	rpcClient.init()
	return rpcClient
}

func (r *RpcClient) init() {
	//todo 暂时写死地址，待改为配置
	//addressList := []string{"127.0.0.1:8091"}
	addressList := []string{"127.0.0.1:8090"}
	if len(addressList) == 0 {
		log.Warn("no have valid seata server list")
	}
	for _, address := range addressList {
		gettyClient := getty.NewTCPClient(
			getty.WithServerAddress(address),
			getty.WithConnectionNumber((int)(r.conf.GettyConfig.ConnectionNum)),
			getty.WithReconnectInterval(r.conf.GettyConfig.ReconnectInterval),
			getty.WithClientTaskPool(gxsync.NewTaskPoolSimple(2)),
		)
		gettyClient.RunEventLoop(r.newSession)
		r.gettyClients = append(r.gettyClients, gettyClient)
	}
}

func (r *RpcClient) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
	)

	if r.conf.GettyConfig.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}

	if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection\n", session.Stat(), session.Conn()))
	}

	tcpConn.SetNoDelay(r.conf.GettyConfig.GettySessionParam.TCPNoDelay)
	tcpConn.SetKeepAlive(r.conf.GettyConfig.GettySessionParam.TCPKeepAlive)
	if r.conf.GettyConfig.GettySessionParam.TCPKeepAlive {
		tcpConn.SetKeepAlivePeriod(r.conf.GettyConfig.GettySessionParam.KeepAlivePeriod)
	}
	tcpConn.SetReadBuffer(r.conf.GettyConfig.GettySessionParam.TCPRBufSize)
	tcpConn.SetWriteBuffer(r.conf.GettyConfig.GettySessionParam.TCPWBufSize)

	session.SetName(r.conf.GettyConfig.GettySessionParam.SessionName)
	session.SetMaxMsgLen(r.conf.GettyConfig.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(rpcPkgHandler)
	session.SetEventListener(r.listener)
	session.SetReadTimeout(r.conf.GettyConfig.GettySessionParam.TCPReadTimeout)
	session.SetWriteTimeout(r.conf.GettyConfig.GettySessionParam.TCPWriteTimeout)
	session.SetCronPeriod((int)(r.conf.GettyConfig.HeartbeatPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(r.conf.GettyConfig.GettySessionParam.WaitTimeout)
	log.Debugf("rpc_client new session:%s", session.Stat())

	return nil
}

func (r *RpcClient) AcquireGettySession() getty.Session {
	// map 遍历是随机的
	var session getty.Session
	r.serverSessionsMap.Range(func(key, value interface{}) bool {
		session = key.(getty.Session)
		if session.IsClosed() {
			r.ReleaseGettySession(session)
		} else {
			return false
		}
		return true
	})
	if session != nil {
		return session
	}
	//if sessionSize == 0 {
	//	ticker := time.NewTicker(time.Duration(CHECK_ALIVE_INTERNAL) * time.Millisecond)
	//	defer ticker.Stop()
	//	for i := 0; i < MAX_CHECK_ALIVE_RETRY; i++ {
	//		<-ticker.C
	//		allSessions.Range(func(key, value interface{}) bool {
	//			session = key.(getty.Session)
	//			if session.IsClosed() {
	//				sessionManager.ReleaseGettySession(session)
	//			} else {
	//				return false
	//			}
	//			return true
	//		})
	//		if session != nil {
	//			return session
	//		}
	//	}
	//}
	return nil
}

func (r *RpcClient) RegisterGettySession(session getty.Session) {
	//r.serverSessionsMap.Store(session, true)
	m, _ := r.serverSessionsMap.LoadOrStore(session.RemoteAddr(), &sync.Map{})
	sMap := m.(*sync.Map)
	sMap.Store(session, true)
	atomic.AddInt32(&r.sessionSize, 1)
}

func (r *RpcClient) ReleaseGettySession(session getty.Session) {
	r.serverSessionsMap.Delete(session)
	if !session.IsClosed() {
		m, _ := r.serverSessionsMap.LoadOrStore(session.RemoteAddr(), &sync.Map{})
		sMap := m.(*sync.Map)
		sMap.Delete(session)
		session.Close()
	}
	atomic.AddInt32(&r.sessionSize, -1)
}
