package server

import (
	"strconv"
	"strings"
	"sync"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/model"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

var (
	// session -> transactionRole
	// TM will register before RM, if a session is not the TM registered,
	// it will be the RM registered
	session_transactionroles = sync.Map{}

	// session -> applicationID
	identified_sessions = sync.Map{}

	// applicationID -> ip -> port -> session
	client_sessions = sync.Map{}

	// applicationID -> resourceIDs
	client_resources = sync.Map{}
)

const (
	ClientIDSplitChar = ":"
	DbkeysSplitChar   = ","
)

type GettySessionManager struct {
	TransactionServiceGroup string
	Version                 string
}

var SessionManager GettySessionManager

func init() {
	SessionManager = GettySessionManager{}
}

func (manager *GettySessionManager) IsRegistered(session getty.Session) bool {
	_, ok := identified_sessions.Load(session)
	return ok
}

func (manager *GettySessionManager) GetRoleFromGettySession(session getty.Session) meta.TransactionRole {
	role, ok := session_transactionroles.Load(session)
	if ok {
		r := role.(meta.TransactionRole)
		return r
	}
	return 0
}

func (manager *GettySessionManager) GetContextFromIdentified(session getty.Session) *RpcContext {
	var applicationID, resourceIDs string

	applicationIDIfc, applicationIDLoaded := identified_sessions.Load(session)
	if applicationIDLoaded {
		applicationID = applicationIDIfc.(string)
	} else {
		return nil
	}

	resourceIDsIfc, resourceIDsLoaded := client_resources.Load(applicationID)
	if resourceIDsLoaded {
		resourceIDs = resourceIDsIfc.(string)
	}

	role := manager.GetRoleFromGettySession(session)

	return NewRpcContext(
		WithRpcContextClientRole(role),
		WithRpcContextApplicationID(applicationID),
		WithRpcContextClientID(buildClientID(applicationID, session)),
		WithRpcContextResourceSet(dbKeyToSet(resourceIDs)),
		WithRpcContextSession(session),
	)
}

func dbKeyToSet(dbKey string) *model.Set {
	if dbKey == "" {
		return nil
	}
	keys := strings.Split(dbKey, DbkeysSplitChar)
	set := model.NewSet()
	for _, key := range keys {
		set.Add(key)
	}
	return set
}

func buildClientID(applicationID string, session getty.Session) string {
	return applicationID + ClientIDSplitChar + session.RemoteAddr()
}

func (manager *GettySessionManager) RegisterTmGettySession(request protocal.RegisterTMRequest, session getty.Session) {
	//todo check version, if not match, refuse to register
	//todo check transaction service group, if not match, refuse to register
	ip := getClientIpFromGettySession(session)
	port := getClientPortFromGettySession(session)

	ipMap, _ := client_sessions.LoadOrStore(request.ApplicationID, &sync.Map{})
	iMap := ipMap.(*sync.Map)
	portMap, _ := iMap.LoadOrStore(ip, &sync.Map{})
	pMap := portMap.(*sync.Map)
	pMap.Store(port, session)

	session_transactionroles.Store(session, meta.TMROLE)
	identified_sessions.Store(session, request.ApplicationID)
}

func (manager *GettySessionManager) RegisterRmGettySession(request protocal.RegisterRMRequest, session getty.Session) {
	//todo check version, if not match, refuse to register
	//todo check transaction service group, if not match, refuse to register
	ip := getClientIpFromGettySession(session)
	port := getClientPortFromGettySession(session)

	ipMap, _ := client_sessions.LoadOrStore(request.ApplicationID, &sync.Map{})
	iMap := ipMap.(*sync.Map)
	portMap, _ := iMap.LoadOrStore(ip, &sync.Map{})
	pMap := portMap.(*sync.Map)
	pMap.Store(port, session)

	session_transactionroles.Store(session, meta.RMROLE)
	identified_sessions.Store(session, request.ApplicationID)
	client_resources.Store(request.ApplicationID, request.ResourceIDs)
}

func (manager *GettySessionManager) GetSameClientGettySession(session getty.Session) getty.Session {
	if !session.IsClosed() {
		return session
	}

	ip := getClientIpFromGettySession(session)
	port := getClientPortFromGettySession(session)

	applicationID, loaded := identified_sessions.Load(session)
	if loaded {
		targetApplicationID := applicationID.(string)
		ipMap, ipMapLoaded := client_sessions.Load(targetApplicationID)
		if ipMapLoaded {
			iMap := ipMap.(*sync.Map)
			portMap, portMapLoaded := iMap.Load(ip)
			if portMapLoaded {
				pMap := portMap.(*sync.Map)
				return getGettySessionFromSamePortMap(pMap, port)
			}
		}
	} else {
		log.Errorf("session {%v} never registered!", session)
	}

	return nil
}

func getGettySessionFromSamePortMap(portMap *sync.Map, exclusivePort int) getty.Session {
	var session getty.Session
	if portMap != nil {
		portMap.Range(func(key interface{}, value interface{}) bool {
			port := key.(int)
			if port == exclusivePort {
				portMap.Delete(key)
				return true
			}

			session = value.(getty.Session)
			if !session.IsClosed() {
				return false
			}
			portMap.Delete(key)
			return true
		})
	}
	return session
}

func (manager *GettySessionManager) GetGettySession(resourceID string, clientID string) (getty.Session, error) {
	var resultSession getty.Session

	clientIDInfo := strings.Split(clientID, ClientIDSplitChar)
	if clientIDInfo == nil || len(clientIDInfo) != 3 {
		return nil, errors.Errorf("Invalid RpcRemoteClient ID: %s", clientID)
	}
	targetApplicationID := clientIDInfo[0]
	targetIP := clientIDInfo[1]
	targetPort, _ := strconv.Atoi(clientIDInfo[2])

	ipMap, ipMapLoaded := client_sessions.Load(targetApplicationID)
	if ipMapLoaded {
		iMap := ipMap.(*sync.Map)
		portMap, portMapLoaded := iMap.Load(targetIP)

		if portMapLoaded {
			pMap := portMap.(*sync.Map)
			session, sessionLoaded := pMap.Load(targetPort)

			// Firstly, try to find the original session through which the branch was registered.
			if sessionLoaded {
				ss := session.(getty.Session)
				if ss.IsClosed() {
					pMap.Delete(targetPort)
					log.Infof("Removed inactive %d", ss)
				} else {
					resultSession = ss
					log.Debugf("Just got exactly the one %v for %s", ss, clientID)
				}
			}

			// The original channel was broken, try another one.
			if resultSession == nil {
				pMap.Range(func(key interface{}, value interface{}) bool {
					ss := value.(getty.Session)

					if ss.IsClosed() {
						pMap.Delete(key)
						log.Infof("Removed inactive %d", ss)
					} else {
						resultSession = ss
						log.Infof("Choose %v on the same IP[%s] as alternative of %s", ss, targetIP, clientID)
						//跳出 range 循环
						return false
					}
					return true
				})
			}
		}

		// No channel on the this cmd node, try another one.
		if resultSession == nil {
			iMap.Range(func(key interface{}, value interface{}) bool {
				ip := key.(string)
				if ip == targetIP {
					return true
				}

				portMapOnOtherIP, _ := value.(*sync.Map)
				if portMapOnOtherIP == nil {
					return true
				}

				portMapOnOtherIP.Range(func(key interface{}, value interface{}) bool {
					ss := value.(getty.Session)

					if ss.IsClosed() {
						portMapOnOtherIP.Delete(key)
						log.Infof("Removed inactive %d", ss)
					} else {
						resultSession = ss
						log.Infof("Choose %v on the same application[%s] as alternative of %s", ss, targetApplicationID, clientID)
						//跳出 range 循环
						return false
					}
					return true
				})

				if resultSession != nil {
					return false
				}
				return true
			})
		}
	}

	if resultSession == nil {
		return nil, errors.New("there is no suitable rpc_client session")
	}

	return resultSession, nil
}

func (manager *GettySessionManager) GetRmSessions() map[string]getty.Session {
	sessions := make(map[string]getty.Session)

	session_transactionroles.Range(func(key interface{}, value interface{}) bool {
		session := key.(getty.Session)
		if session.IsClosed() {
			session_transactionroles.Delete(key)
		}
		return true
	})

	client_sessions.Range(func(key, value interface{}) bool {
		applicationID := key.(string)
		ipMap := value.(*sync.Map)
		session := getRMGettySessionFromIpMap(ipMap)

		resourceIDs, loaded := client_resources.Load(applicationID)
		if loaded {
			rscIDs := resourceIDs.(string)
			dbKeySet := dbKeyToSet(rscIDs)
			resources := dbKeySet.List()
			for _, resourceID := range resources {
				sessions[resourceID] = session
			}
		}
		return true
	})

	return sessions
}

func getRMGettySessionFromIpMap(ipMap *sync.Map) getty.Session {
	var chosenSession getty.Session

	ipMap.Range(func(key interface{}, value interface{}) bool {
		portMap := value.(*sync.Map)
		portMap.Range(func(key interface{}, value interface{}) bool {
			session := key.(getty.Session)
			if session.IsClosed() {
				portMap.Delete(key)
				log.Infof("Removed inactive %d", session)
			} else {
				role, loaded := session_transactionroles.Load(session)
				if loaded {
					r := role.(meta.TransactionRole)
					if r != meta.TMROLE {
						chosenSession = session
						return false
					}
				}
			}
			return true
		})
		if chosenSession != nil {
			return false
		}
		return true
	})
	return chosenSession
}

func getClientIpFromGettySession(session getty.Session) string {
	clientIp := session.RemoteAddr()
	if strings.Contains(clientIp, IpPortSplitChar) {
		idx := strings.Index(clientIp, IpPortSplitChar)
		clientIp = clientIp[:idx]
	}
	return clientIp
}

func getClientPortFromGettySession(session getty.Session) int {
	address := session.RemoteAddr()
	port := 0
	if strings.Contains(address, IpPortSplitChar) {
		idx := strings.LastIndex(address, IpPortSplitChar)
		port, _ = strconv.Atoi(address[idx+1:])
	}
	return port
}

func (manager *GettySessionManager) ReleaseGettySession(session getty.Session) {
	session_transactionroles.Delete(session)
	identified_sessions.Delete(session)
}
