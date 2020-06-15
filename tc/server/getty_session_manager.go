package server

import (
	"strconv"
	"strings"
	"sync"

	"github.com/dubbogo/getty"
	"github.com/pkg/errors"
	"github.com/xiaobudongzhang/seata-golang/base/meta"
	"github.com/xiaobudongzhang/seata-golang/base/model"
	"github.com/xiaobudongzhang/seata-golang/base/protocal"
	"github.com/xiaobudongzhang/seata-golang/pkg/logging"
)

var (
	/**
	 * resourceId -> applicationId -> ip -> port -> RpcContext
	 */
	rm_sessions = sync.Map{}

	/**
	 * ip+appname -> port -> RpcContext
	 */
	tm_sessions = sync.Map{}
)

const (
	ClientIdSplitChar = ":"
	DbkeysSplitChar   = ","
)

type GettySessionManager struct {
	IdentifiedSessions *sync.Map
}

var SessionManager GettySessionManager

func init() {
	SessionManager = GettySessionManager{IdentifiedSessions: &sync.Map{}}
}

func (manager *GettySessionManager) IsRegistered(session getty.Session) bool {
	_, ok := manager.IdentifiedSessions.Load(session)
	return ok
}

func (manager *GettySessionManager) GetRoleFromGettySession(session getty.Session) meta.TransactionRole {
	context, ok := manager.IdentifiedSessions.Load(session)
	if ok {
		return context.(*RpcContext).ClientRole
	}
	return 0
}

func (manager *GettySessionManager) GetContextFromIdentified(session getty.Session) *RpcContext {
	context, ok := manager.IdentifiedSessions.Load(session)
	if ok {
		rpcContext := context.(*RpcContext)
		return rpcContext
	}
	return nil
}

func (manager *GettySessionManager) RegisterTmGettySession(request protocal.RegisterTMRequest, session getty.Session) {
	//todo check version
	rpcContext := buildGettySessionHolder(meta.TMROLE, request.Version, request.ApplicationId, request.TransactionServiceGroup, "", session)
	rpcContext.HoldInIdentifiedGettySessions(manager.IdentifiedSessions)
	clientIdentified := rpcContext.ApplicationId + ClientIdSplitChar + getClientIpFromGettySession(session)
	clientIdentifiedMap, _ := tm_sessions.LoadOrStore(clientIdentified, &sync.Map{})
	cMap := clientIdentifiedMap.(*sync.Map)
	rpcContext.HoldInClientGettySessions(cMap)
}

func (manager *GettySessionManager) RegisterRmGettySession(resourceManagerRequest protocal.RegisterRMRequest, session getty.Session) {
	//todo check version
	var rpcContext *RpcContext
	dbKeySet := dbKeyToSet(resourceManagerRequest.ResourceIds)
	context, ok := manager.IdentifiedSessions.Load(session)
	if ok {
		rpcContext = context.(*RpcContext)
		rpcContext.AddResources(dbKeySet)
	} else {
		rpcContext = buildGettySessionHolder(meta.RMROLE, resourceManagerRequest.Version, resourceManagerRequest.ApplicationId,
			resourceManagerRequest.TransactionServiceGroup, resourceManagerRequest.ResourceIds, session)
		rpcContext.HoldInIdentifiedGettySessions(manager.IdentifiedSessions)
	}
	if dbKeySet == nil || dbKeySet.IsEmpty() {
		return
	}
	for _, resourceId := range dbKeySet.List() {
		applicationMap, _ := rm_sessions.LoadOrStore(resourceId, &sync.Map{})
		aMap, _ := applicationMap.(*sync.Map)
		ipMap, _ := aMap.LoadOrStore(resourceManagerRequest.ApplicationId, &sync.Map{})
		iMap, _ := ipMap.(*sync.Map)
		clientIp := getClientIpFromGettySession(session)
		portMap, _ := iMap.LoadOrStore(clientIp, &sync.Map{})
		pMap, _ := portMap.(*sync.Map)

		rpcContext.HoldInResourceManagerGettySessions(resourceId, pMap)
		// 老实讲，我不知道为什么要写这么一个方法，双重保证？
		manager.updateGettySessionsResource(resourceId, clientIp, resourceManagerRequest.ApplicationId)
	}
}

func (manager *GettySessionManager) updateGettySessionsResource(resourceId string, clientIp string, applicationId string) {
	applicationMap, _ := rm_sessions.Load(resourceId)
	aMap, _ := applicationMap.(*sync.Map)
	ipMap, _ := aMap.Load(applicationId)
	iMap, _ := ipMap.(*sync.Map)
	portMap, _ := iMap.Load(clientIp)
	pMap, _ := portMap.(*sync.Map)

	rm_sessions.Range(func(key interface{}, value interface{}) bool {
		resourceKey, ok := key.(string)
		if ok && resourceKey != resourceId {
			appMap, _ := value.(*sync.Map)

			clientIpMap, clientIpMapLoaded := appMap.Load(applicationId)
			if clientIpMapLoaded {
				cipMap, _ := clientIpMap.(*sync.Map)
				clientPortMap, clientPortMapLoaded := cipMap.Load(clientIp)
				if clientPortMapLoaded {
					cpMap := clientPortMap.(*sync.Map)
					cpMap.Range(func(key interface{}, value interface{}) bool {
						port, _ := key.(int)
						rpcContext, _ := value.(*RpcContext)
						_, ok := pMap.LoadOrStore(port, rpcContext)
						if ok {
							rpcContext.HoldInResourceManagerGettySessionsWithoutPortMap(resourceId, port)
						}
						return true
					})
				}
			}
		}
		return true
	})
}

func (manager *GettySessionManager) GetSameClientGettySession(session getty.Session) getty.Session {
	if !session.IsClosed() {
		return session
	}

	rpcContext := manager.GetContextFromIdentified(session)
	if rpcContext == nil {
		logging.Logger.Errorf("rpcContext is null,channel:{%v},active:{%t}", session, !session.IsClosed())
	}
	if !rpcContext.session.IsClosed() {
		return rpcContext.session
	}

	clientPort := getClientPortFromGettySession(session)
	if rpcContext.ClientRole == meta.TMROLE {
		clientIdentified := rpcContext.ApplicationId + ClientIdSplitChar + getClientIpFromGettySession(session)
		clientRpcMap, ok := tm_sessions.Load(clientIdentified)
		if !ok {
			return nil
		}
		clientMap := clientRpcMap.(*sync.Map)
		return getGettySessionFromSameClientMap(clientMap, clientPort)
	} else if rpcContext.ClientRole == meta.RMROLE {
		var sameClientSession getty.Session
		rpcContext.ClientRMHolderMap.Range(func(key interface{}, value interface{}) bool {
			clientRmMap := value.(*sync.Map)
			sameClientSession = getGettySessionFromSameClientMap(clientRmMap, clientPort)
			if sameClientSession != nil {
				return false
			}
			return true
		})
		return sameClientSession
	}
	return nil
}

func getGettySessionFromSameClientMap(clientGettySessionMap *sync.Map, exclusivePort int) getty.Session {
	var session getty.Session
	if clientGettySessionMap != nil {
		clientGettySessionMap.Range(func(key interface{}, value interface{}) bool {
			port, ok := key.(int)
			if ok {
				if port == exclusivePort {
					clientGettySessionMap.Delete(key)
					return true
				}
			}

			context := value.(*RpcContext)
			session = context.session
			if !session.IsClosed() {
				return false
			}
			clientGettySessionMap.Delete(key)
			return true
		})
	}
	return session
}

func (manager *GettySessionManager) GetGettySession(resourceId string, clientId string) (getty.Session, error) {
	var resultSession getty.Session

	clientIdInfo := strings.Split(clientId, ClientIdSplitChar)
	if clientIdInfo == nil || len(clientIdInfo) != 3 {
		return nil, errors.Errorf("Invalid RpcRemoteClient ID:%d", clientId)
	}
	targetApplicationId := clientIdInfo[0]
	targetIP := clientIdInfo[1]
	targetPort, _ := strconv.Atoi(clientIdInfo[2])

	applicationMap, ok := rm_sessions.Load(resourceId)
	if targetApplicationId == "" || !ok || applicationMap == nil {
		logging.Logger.Infof("No channel is available for resource[%s]", resourceId)
	}
	appMap, _ := applicationMap.(*sync.Map)

	clientIpMap, clientIpMapLoaded := appMap.Load(targetApplicationId)
	if clientIpMapLoaded {
		ipMap, _ := clientIpMap.(*sync.Map)

		portMap, portMapLoaded := ipMap.Load(targetIP)
		if portMapLoaded {
			pMap, _ := portMap.(*sync.Map)
			context, contextLoaded := pMap.Load(targetPort)
			// Firstly, try to find the original channel through which the branch was registered.
			if contextLoaded {
				rpcContext := context.(*RpcContext)
				if !rpcContext.session.IsClosed() {
					resultSession = rpcContext.session
					logging.Logger.Debugf("Just got exactly the one %v for %s", rpcContext.session, clientId)
				} else {
					pMap.Delete(targetPort)
					logging.Logger.Infof("Removed inactive %d", rpcContext.session)
				}
			}

			// The original channel was broken, try another one.
			if resultSession == nil {
				pMap.Range(func(key interface{}, value interface{}) bool {
					rpcContext := value.(*RpcContext)

					if !rpcContext.session.IsClosed() {
						resultSession = rpcContext.session
						logging.Logger.Infof("Choose %v on the same IP[%s] as alternative of %s", rpcContext.session, targetIP, clientId)
						//跳出 range 循环
						return false
					} else {
						pMap.Delete(key)
						logging.Logger.Infof("Removed inactive %d", rpcContext.session)
					}
					return true
				})
			}
		}

		// No channel on the this app node, try another one.
		if resultSession == nil {
			ipMap.Range(func(key interface{}, value interface{}) bool {
				ip := key.(string)
				if ip == targetIP {
					return true
				}

				portMapOnOtherIP, _ := value.(*sync.Map)
				if portMapOnOtherIP == nil {
					return true
				}

				portMapOnOtherIP.Range(func(key interface{}, value interface{}) bool {
					rpcContext := value.(*RpcContext)

					if !rpcContext.session.IsClosed() {
						resultSession = rpcContext.session
						logging.Logger.Infof("Choose %v on the same application[%s] as alternative of %s", rpcContext.session, targetApplicationId, clientId)
						//跳出 range 循环
						return false
					} else {
						portMapOnOtherIP.Delete(key)
						logging.Logger.Infof("Removed inactive %d", rpcContext.session)
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
		resultSession = tryOtherApp(appMap, targetApplicationId)
		if resultSession == nil {
			logging.Logger.Infof("No channel is available for resource[%s] as alternative of %s", resourceId, clientId)
		} else {
			logging.Logger.Infof("Choose %v on the same resource[%s] as alternative of %s", resultSession, resourceId, clientId)
		}
	}
	return resultSession, nil
}

func tryOtherApp(applicationMap *sync.Map, myApplicationId string) getty.Session {
	var chosenChannel getty.Session
	applicationMap.Range(func(key interface{}, value interface{}) bool {
		applicationId := key.(string)
		if myApplicationId != "" && applicationId == myApplicationId {
			return true
		}

		targetIPMap, _ := value.(*sync.Map)
		targetIPMap.Range(func(key interface{}, value interface{}) bool {
			if value == nil {
				return true
			}
			portMap, _ := value.(*sync.Map)

			portMap.Range(func(key interface{}, value interface{}) bool {
				rpcContext := value.(*RpcContext)
				if !rpcContext.session.IsClosed() {
					chosenChannel = rpcContext.session
					return false
				} else {
					portMap.Delete(key)
					logging.Logger.Infof("Removed inactive %d", rpcContext.session)
				}
				return true
			})
			if chosenChannel != nil {
				return false
			}
			return true
		})
		if chosenChannel != nil {
			return false
		}
		return true
	})
	return chosenChannel
}

func buildGettySessionHolder(role meta.TransactionRole, version string, applicationId string,
	txServiceGroup string, dbKeys string, session getty.Session) *RpcContext {
	return &RpcContext{
		ClientRole:              role,
		Version:                 version,
		ApplicationId:           applicationId,
		TransactionServiceGroup: txServiceGroup,
		ClientId:                buildClientId(applicationId, session),
		session:                 session,
		ResourceSets:            dbKeyToSet(dbKeys),
	}
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

func buildClientId(applicationId string, session getty.Session) string {
	return applicationId + ClientIdSplitChar + session.RemoteAddr()
}

func (manager *GettySessionManager) GetRmSessions() map[string]getty.Session {
	sessions := make(map[string]getty.Session)
	rm_sessions.Range(func(key interface{}, value interface{}) bool {
		resourceId, _ := key.(string)
		applicationMap := value.(*sync.Map)
		session := tryOtherApp(applicationMap, "")
		if session == nil {
			return false
		}
		sessions[resourceId] = session
		return true
	})
	return sessions
}
