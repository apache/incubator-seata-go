package rpc_client

import (
	"sync"
	"sync/atomic"
	"time"

	getty "github.com/apache/dubbo-getty"
)

var (
	MAX_CHECK_ALIVE_RETRY = 600

	CHECK_ALIVE_INTERNAL = 100

	//allSessions = sync.Map{}

	// serverAddress -> rpc_client.Session -> bool
	//serverSessions = sync.Map{}

	sessionSize int32 = 0

	clientSessionManager = &GettyClientSessionManager{}
)

type GettyClientSessionManager struct {
	allSessions    sync.Map
	serverSessions sync.Map
}

func (sessionManager *GettyClientSessionManager) AcquireGettySession() getty.Session {
	// map 遍历是随机的
	var session getty.Session
	sessionManager.allSessions.Range(func(key, value interface{}) bool {
		session = key.(getty.Session)
		if session.IsClosed() {
			sessionManager.ReleaseGettySession(session)
		} else {
			return false
		}
		return true
	})
	if session != nil {
		return session
	}
	if sessionSize == 0 {
		ticker := time.NewTicker(time.Duration(CHECK_ALIVE_INTERNAL) * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < MAX_CHECK_ALIVE_RETRY; i++ {
			<-ticker.C
			sessionManager.allSessions.Range(func(key, value interface{}) bool {
				session = key.(getty.Session)
				if session.IsClosed() {
					sessionManager.ReleaseGettySession(session)
				} else {
					return false
				}
				return true
			})
			if session != nil {
				return session
			}
		}
	}
	return nil
}

func (sessionManager *GettyClientSessionManager) AcquireGettySessionByServerAddress(serverAddress string) getty.Session {
	m, _ := sessionManager.serverSessions.LoadOrStore(serverAddress, &sync.Map{})
	sMap := m.(*sync.Map)

	var session getty.Session
	sMap.Range(func(key, value interface{}) bool {
		session = key.(getty.Session)
		if session.IsClosed() {
			sessionManager.ReleaseGettySession(session)
		} else {
			return false
		}
		return true
	})
	return session
}

func (sessionManager *GettyClientSessionManager) ReleaseGettySession(session getty.Session) {
	sessionManager.allSessions.Delete(session)
	if !session.IsClosed() {
		m, _ := sessionManager.serverSessions.LoadOrStore(session.RemoteAddr(), &sync.Map{})
		sMap := m.(*sync.Map)
		sMap.Delete(session)
		session.Close()
	}
	atomic.AddInt32(&sessionSize, -1)
}

func (sessionManager *GettyClientSessionManager) RegisterGettySession(session getty.Session) {
	sessionManager.allSessions.Store(session, true)
	m, _ := sessionManager.serverSessions.LoadOrStore(session.RemoteAddr(), &sync.Map{})
	sMap := m.(*sync.Map)
	sMap.Store(session, true)
	atomic.AddInt32(&sessionSize, 1)
}
