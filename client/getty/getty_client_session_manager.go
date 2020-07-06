package getty

import (
	"sync"
	"time"
)

import (
	"github.com/dubbogo/getty"
)

var (
	MAX_CHECK_ALIVE_RETRY = 600

	CHECK_ALIVE_INTERNAL = 100

	// string -> getty_session
	sessions = sync.Map{}

	clientSessionManager = &GettyClientSessionManager{}
)

type GettyClientSessionManager struct {
}

func (sessionManager *GettyClientSessionManager) AcquireGettySession(serverAddress string) getty.Session {
	sessionToServer, ok := sessions.Load(serverAddress)
	if ok {
		session := sessionToServer.(getty.Session)
		if !session.IsClosed() {
			return session
		}
	}
	ticker := time.NewTicker(time.Duration(CHECK_ALIVE_INTERNAL) * time.Millisecond)
	defer ticker.Stop()
	for i := 0; i < MAX_CHECK_ALIVE_RETRY; i++ {
		<-ticker.C
		sessionToServer, ok := sessions.Load(serverAddress)
		if ok {
			session := sessionToServer.(getty.Session)
			if !session.IsClosed() {
				return session
			}
		}
	}
	return nil
}

func (sessionManager *GettyClientSessionManager) ReleaseGettySession(session getty.Session, serverAddress string) {
	if session == nil && serverAddress == "" {
		return
	}
	ss, ok := sessions.Load(serverAddress)
	if ok && ss == session {
		sessions.Delete(serverAddress)
	}
	session.Close()
}

func (sessionManager *GettyClientSessionManager) RegisterGettySession(session getty.Session, serverAddress string) {
	//sessionToServer,ok := sessions.Load(serverAddress)
	//if ok {
	//	session := sessionToServer.(getty.Session)
	//	if !session.IsClosed() {
	//		return
	//	}
	//}
	sessions.Store(serverAddress, session)
}
