package getty

import (
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/getty"
)

var (
	MAX_CHECK_ALIVE_RETRY = 600

	CHECK_ALIVE_INTERNAL = 100

	sessions = sync.Map{}

	sessionSize int32 = 0

	clientSessionManager = &GettyClientSessionManager{}
)

type GettyClientSessionManager struct {
}

func (sessionManager *GettyClientSessionManager) AcquireGettySession() getty.Session {
	// map 遍历是随机的
	var session getty.Session
	sessions.Range(func(key, value interface{}) bool {
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
			sessions.Range(func(key, value interface{}) bool {
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
	ss := sessionManager.AcquireGettySession()
	if ss != nil {
		if ss.RemoteAddr() == serverAddress {
			return ss
		} else {
			return sessionManager.AcquireGettySessionByServerAddress(serverAddress)
		}
	}
	return nil
}

func (sessionManager *GettyClientSessionManager) ReleaseGettySession(session getty.Session) {
	sessions.Delete(session)
	atomic.AddInt32(&sessionSize, -1)
	session.Close()
}

func (sessionManager *GettyClientSessionManager) RegisterGettySession(session getty.Session) {
	sessions.Store(session, true)
	atomic.AddInt32(&sessionSize, 1)
}
