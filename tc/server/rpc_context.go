package server

import (
	"errors"
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/base/model"
	"github.com/dubbogo/getty"
	"strconv"
	"strings"
	"sync"
)

const IpPortSplitChar = ":"

type RpcContext struct {
	ClientRole              meta.TransactionRole
	Version                 string
	ApplicationId           string
	TransactionServiceGroup string
	ClientId                string
	session                 getty.Session
	ResourceSets            *model.Set

	/**
	 * <getty.Session,*RpcContext>
	 */
	ClientIDHolderMap *sync.Map

	/**
	 * <int,RpcContext>
	 */
	ClientTMHolderMap *sync.Map

	/**
	 * resourceId -> int -> RpcContext>
	 */
	ClientRMHolderMap *sync.Map
}

func (context *RpcContext) Release() {
	clientPort := getClientPortFromGettySession(context.session)
	if context.ClientIDHolderMap != nil {
		context.ClientIDHolderMap = nil
	}
	if context.ClientRole == meta.TMROLE && context.ClientTMHolderMap != nil {
		context.ClientTMHolderMap.Delete(clientPort)
		context.ClientTMHolderMap = nil
	}
	if context.ClientRole == meta.RMROLE && context.ClientRMHolderMap != nil {
		context.ClientRMHolderMap.Range(func (key interface{}, value interface{}) bool {
			m := value.(*sync.Map)
			m.Delete(clientPort)
			return true
		})
		context.ClientRMHolderMap = nil
	}
	if context.ResourceSets != nil {
		context.ResourceSets.Clear()
	}
}

func (context *RpcContext) HoldInClientGettySessions(clientTMHolderMap *sync.Map) error {
	if context.ClientTMHolderMap != nil {
		return errors.New("illegal state")
	}
	context.ClientTMHolderMap = clientTMHolderMap
	clientPort := getClientPortFromGettySession(context.session)
	context.ClientTMHolderMap.Store(clientPort,context)
	return nil
}

func (context *RpcContext) HoldInIdentifiedGettySessions(clientIDHolderMap *sync.Map) error {
	if context.ClientIDHolderMap != nil {
		return errors.New("illegal state")
	}
	context.ClientIDHolderMap = clientIDHolderMap
	context.ClientIDHolderMap.Store(context.session,context)
	return nil
}

func (context *RpcContext) HoldInResourceManagerGettySessions(resourceId string,portMap *sync.Map) {
	if context.ClientRMHolderMap == nil {
		context.ClientRMHolderMap = &sync.Map{}
	}
	clientPort := getClientPortFromGettySession(context.session)
	portMap.Store(clientPort,context)
	context.ClientRMHolderMap.Store(resourceId,portMap)
}

func (context *RpcContext) HoldInResourceManagerGettySessionsWithoutPortMap(resourceId string,clientPort int) {
	if context.ClientRMHolderMap == nil {
		context.ClientRMHolderMap = &sync.Map{}
	}
	portMap,_ := context.ClientRMHolderMap.LoadOrStore(resourceId,&sync.Map{})
	pm := portMap.(*sync.Map)
	pm.Store(clientPort,context)
}

func (context *RpcContext) AddResource(resource string) {
	if resource != "" {
		if context.ResourceSets == nil {
			context.ResourceSets = model.NewSet()
		}
		context.ResourceSets.Add(resource)
	}
}

func (context *RpcContext) AddResources(resources *model.Set) {
	if resources != nil {
		if context.ResourceSets == nil {
			context.ResourceSets = model.NewSet()
		}
		for _,resource := range resources.List() {
			context.ResourceSets.Add(resource)
		}
	}
}

func getClientIpFromGettySession(session getty.Session) string {
	clientIp := session.RemoteAddr()
	if strings.Contains(clientIp,IpPortSplitChar) {
		idx := strings.Index(clientIp,IpPortSplitChar)
		clientIp = clientIp[:idx]
	}
	return clientIp
}

func getClientPortFromGettySession(session getty.Session) int {
	address := session.RemoteAddr()
	port := 0
	if strings.Contains(address,IpPortSplitChar) {
		idx := strings.LastIndex(address,IpPortSplitChar)
		port,_ = strconv.Atoi(address[idx+1:])
	}
	return port
}