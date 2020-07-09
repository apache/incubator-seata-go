package getty

import (
	"time"
)

import (
	"github.com/dubbogo/getty"
)

var (
	MAX_CHECK_ALIVE_RETRY = 600

	CHECK_ALIVE_INTERNAL = 100

	sessions = make(map[getty.Session]bool)

	clientSessionManager = &GettyClientSessionManager{}
)

type GettyClientSessionManager struct {
}

func (sessionManager *GettyClientSessionManager) AcquireGettySession() getty.Session {
	// map 遍历是随机的
	for session := range sessions {
		if session.IsClosed() {
			sessionManager.ReleaseGettySession(session)
		} else {
			return session
		}
	}
	if len(sessions) == 0 {
		ticker := time.NewTicker(time.Duration(CHECK_ALIVE_INTERNAL) * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < MAX_CHECK_ALIVE_RETRY; i++ {
			<-ticker.C
			for session := range sessions {
				if session.IsClosed() {
					sessionManager.ReleaseGettySession(session)
				} else {
					return session
				}
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
	delete(sessions, session)
	session.Close()
}

func (sessionManager *GettyClientSessionManager) RegisterGettySession(session getty.Session) {
	sessions[session] = true
}
