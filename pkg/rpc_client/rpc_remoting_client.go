package rpc_client

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"

	gxtime "github.com/dubbogo/gost/time"

	"github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"github.com/seata/seata-go/pkg/config"
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/protocol/codec"
	log "github.com/seata/seata-go/pkg/util/log"
	"github.com/seata/seata-go/pkg/util/runtime"
)

const (
	RPC_REQUEST_TIMEOUT = 30 * time.Second
)

var rpcRemoteClient *RpcRemoteClient

func InitRpcRemoteClient() *RpcRemoteClient {
	rpcRemoteClient = &RpcRemoteClient{
		conf:                         config.GetClientConfig(),
		idGenerator:                  &atomic.Uint32{},
		futures:                      &sync.Map{},
		mergeMsgMap:                  &sync.Map{},
		rpcMessageChannel:            make(chan protocol.RpcMessage, 100),
		BranchRollbackRequestChannel: make(chan RpcRMMessage),
		BranchCommitRequestChannel:   make(chan RpcRMMessage),
		GettySessionOnOpenChannel:    make(chan string),
	}
	if rpcRemoteClient.conf.EnableClientBatchSendRequest {
		go rpcRemoteClient.processMergedMessage()
	}
	return rpcRemoteClient
}

func GetRpcRemoteClient() *RpcRemoteClient {
	return rpcRemoteClient
}

type RpcRemoteClient struct {
	conf                         *config.ClientConfig
	idGenerator                  *atomic.Uint32
	futures                      *sync.Map
	mergeMsgMap                  *sync.Map
	rpcMessageChannel            chan protocol.RpcMessage
	BranchCommitRequestChannel   chan RpcRMMessage
	BranchRollbackRequestChannel chan RpcRMMessage
	GettySessionOnOpenChannel    chan string
}

// OnOpen ...
func (client *RpcRemoteClient) OnOpen(session getty.Session) error {
	go func() {
		request := protocol.RegisterTMRequest{AbstractIdentifyRequest: protocol.AbstractIdentifyRequest{
			Version:                 client.conf.SeataVersion,
			ApplicationId:           client.conf.ApplicationID,
			TransactionServiceGroup: client.conf.TransactionServiceGroup,
		}}
		_, err := client.sendAsyncRequestWithResponse(session, request, RPC_REQUEST_TIMEOUT)
		if err == nil {
			clientSessionManager.RegisterGettySession(session)
			client.GettySessionOnOpenChannel <- session.RemoteAddr()
		}
	}()

	return nil
}

// OnError ...
func (client *RpcRemoteClient) OnError(session getty.Session, err error) {
	clientSessionManager.ReleaseGettySession(session)
}

// OnClose ...
func (client *RpcRemoteClient) OnClose(session getty.Session) {
	clientSessionManager.ReleaseGettySession(session)
}

// OnMessage ...
func (client *RpcRemoteClient) OnMessage(session getty.Session, pkg interface{}) {
	log.Debugf("received message: {%#v}", pkg)
	rpcMessage, ok := pkg.(protocol.RpcMessage)
	if !ok {
		log.Errorf("received message is not protocol.RpcMessage. pkg: %#v", pkg)
		return
	}

	heartBeat, isHeartBeat := rpcMessage.Body.(protocol.HeartBeatMessage)
	if isHeartBeat && heartBeat == protocol.HeartBeatMessagePong {
		log.Debugf("received PONG from %s", session.RemoteAddr())
		return
	}

	if rpcMessage.MessageType == protocol.MSGTypeRequest ||
		rpcMessage.MessageType == protocol.MSGTypeRequestOneway {
		log.Debugf("msgID: %d, body: %#v", rpcMessage.ID, rpcMessage.Body)

		client.onMessage(rpcMessage, session.RemoteAddr())
		return
	}

	mergedResult, isMergedResult := rpcMessage.Body.(protocol.MergeResultMessage)
	if isMergedResult {
		mm, loaded := client.mergeMsgMap.Load(rpcMessage.ID)
		if loaded {
			mergedMessage := mm.(protocol.MergedWarpMessage)
			log.Infof("rpcMessageID: %d, rpcMessage: %#v, result: %#v", rpcMessage.ID, mergedMessage, mergedResult)
			for i := 0; i < len(mergedMessage.Msgs); i++ {
				msgID := mergedMessage.MsgIds[i]
				resp, loaded := client.futures.Load(msgID)
				if loaded {
					response := resp.(*protocol.MessageFuture)
					response.Response = mergedResult.Msgs[i]
					response.Done <- true
					client.futures.Delete(msgID)
				}
			}
			client.mergeMsgMap.Delete(rpcMessage.ID)
		}
	} else {
		resp, loaded := client.futures.Load(rpcMessage.ID)
		if loaded {
			response := resp.(*protocol.MessageFuture)
			response.Response = rpcMessage.Body
			response.Done <- true
			client.futures.Delete(rpcMessage.ID)
		}
	}
}

// OnCron ...
func (client *RpcRemoteClient) OnCron(session getty.Session) {
	client.defaultSendRequest(session, protocol.HeartBeatMessagePing)
}

func (client *RpcRemoteClient) onMessage(rpcMessage protocol.RpcMessage, serverAddress string) {
	msg := rpcMessage.Body.(protocol.MessageTypeAware)
	log.Debugf("onMessage: %#v", msg)
	switch msg.GetTypeCode() {
	case protocol.TypeBranchCommit:
		client.BranchCommitRequestChannel <- RpcRMMessage{
			RpcMessage:    rpcMessage,
			ServerAddress: serverAddress,
		}
	case protocol.TypeBranchRollback:
		client.BranchRollbackRequestChannel <- RpcRMMessage{
			RpcMessage:    rpcMessage,
			ServerAddress: serverAddress,
		}
	case protocol.TypeRmDeleteUndolog:
		break
	default:
		break
	}
}

//*************************************
// ClientMessageSender
//*************************************
func (client *RpcRemoteClient) SendMsgWithResponse(msg interface{}) (interface{}, error) {
	if client.conf.EnableClientBatchSendRequest {
		return client.sendAsyncRequest2(msg, RPC_REQUEST_TIMEOUT)
	}
	return client.SendMsgWithResponseAndTimeout(msg, RPC_REQUEST_TIMEOUT)
}

func (client *RpcRemoteClient) SendMsgWithResponseAndTimeout(msg interface{}, timeout time.Duration) (interface{}, error) {
	ss := clientSessionManager.AcquireGettySession()
	return client.sendAsyncRequestWithResponse(ss, msg, timeout)
}

func (client *RpcRemoteClient) SendResponse(request protocol.RpcMessage, serverAddress string, msg interface{}) {
	client.defaultSendResponse(request, clientSessionManager.AcquireGettySessionByServerAddress(serverAddress), msg)
}

func (client *RpcRemoteClient) sendAsyncRequestWithResponse(session getty.Session, msg interface{}, timeout time.Duration) (interface{}, error) {
	if timeout <= time.Duration(0) {
		return nil, errors.New("timeout should more than 0ms")
	}
	return client.sendAsyncRequest(session, msg, timeout)
}

func (client *RpcRemoteClient) sendAsyncRequestWithoutResponse(session getty.Session, msg interface{}) error {
	_, err := client.sendAsyncRequest(session, msg, time.Duration(0))
	return err
}

func (client *RpcRemoteClient) sendAsyncRequest(session getty.Session, msg interface{}, timeout time.Duration) (interface{}, error) {
	var err error
	if session == nil || session.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	rpcMessage := protocol.RpcMessage{
		ID:          int32(client.idGenerator.Inc()),
		MessageType: protocol.MSGTypeRequestOneway,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	resp := protocol.NewMessageFuture(rpcMessage)
	client.futures.Store(rpcMessage.ID, resp)
	//config timeout
	_, _, err = session.WritePkg(rpcMessage, time.Duration(0))
	if err != nil {
		client.futures.Delete(rpcMessage.ID)
		log.Errorf("send message: %#v, session: %s", rpcMessage, session.Stat())
		return nil, err
	}

	log.Debugf("send message: %#v, session: %s", rpcMessage, session.Stat())

	if timeout > time.Duration(0) {
		select {
		case <-gxtime.GetDefaultTimerWheel().After(timeout):
			client.futures.Delete(rpcMessage.ID)
			if session != nil {
				return nil, errors.Errorf("wait response timeout, ip: %s, request: %#v", session.RemoteAddr(), rpcMessage)
			} else {
				return nil, errors.Errorf("wait response timeout and session is nil, request: %#v", rpcMessage)
			}
		case <-resp.Done:
			err = resp.Err
			return resp.Response, err
		}
	}

	return nil, err
}

func (client *RpcRemoteClient) sendAsyncRequest2(msg interface{}, timeout time.Duration) (interface{}, error) {
	var err error
	rpcMessage := protocol.RpcMessage{
		ID:          int32(client.idGenerator.Inc()),
		MessageType: protocol.MSGTypeRequest,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	resp := protocol.NewMessageFuture(rpcMessage)
	client.futures.Store(rpcMessage.ID, resp)

	client.rpcMessageChannel <- rpcMessage

	log.Infof("send message: %#v", rpcMessage)

	if timeout > time.Duration(0) {
		select {
		case <-gxtime.GetDefaultTimerWheel().After(timeout):
			client.futures.Delete(rpcMessage.ID)
			return nil, errors.Errorf("wait response timeout, request: %#v", rpcMessage)
		case <-resp.Done:
			err = resp.Err
		}
		return resp.Response, err
	}
	return nil, err
}

func (client *RpcRemoteClient) sendAsync(session getty.Session, msg interface{}) error {
	var err error
	if session == nil || session.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	rpcMessage := protocol.RpcMessage{
		ID:          int32(client.idGenerator.Inc()),
		MessageType: protocol.MSGTypeRequestOneway,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	log.Infof("store message, id %d : %#v", rpcMessage.ID, msg)
	client.mergeMsgMap.Store(rpcMessage.ID, msg)
	//config timeout
	pkgLen, sendLen, err := session.WritePkg(rpcMessage, time.Duration(0))
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		runtime.GoWithRecover(func() {
			session.Close()
		}, nil)
		return errors.Wrap(err, "pkg not send completely!")
	}
	return nil
}

func (client *RpcRemoteClient) defaultSendRequest(session getty.Session, msg interface{}) {
	rpcMessage := protocol.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Codec:      codec.SEATA,
		Compressor: 0,
		Body:       msg,
	}
	_, ok := msg.(protocol.HeartBeatMessage)
	if ok {
		rpcMessage.MessageType = protocol.MSGTypeHeartbeatRequest
	} else {
		rpcMessage.MessageType = protocol.MSGTypeRequest
	}
	pkgLen, sendLen, err := session.WritePkg(rpcMessage, client.conf.GettyConfig.GettySessionParam.TCPWriteTimeout)
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		runtime.GoWithRecover(func() {
			session.Close()
		}, nil)
	}
}

func (client *RpcRemoteClient) defaultSendResponse(request protocol.RpcMessage, session getty.Session, msg interface{}) {
	resp := protocol.RpcMessage{
		ID:         request.ID,
		Codec:      request.Codec,
		Compressor: request.Compressor,
		Body:       msg,
	}
	_, ok := msg.(protocol.HeartBeatMessage)
	if ok {
		resp.MessageType = protocol.MSGTypeHeartbeatResponse
	} else {
		resp.MessageType = protocol.MSGTypeResponse
	}

	pkgLen, sendLen, err := session.WritePkg(resp, time.Duration(0))
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		runtime.GoWithRecover(func() {
			session.Close()
		}, nil)
	}
}

func (client *RpcRemoteClient) RegisterResource(serverAddress string, request protocol.RegisterRMRequest) {
	session := clientSessionManager.AcquireGettySessionByServerAddress(serverAddress)
	if session != nil {
		err := client.sendAsyncRequestWithoutResponse(session, request)
		if err != nil {
			log.Errorf("register resource failed, session:{},resourceID:{}", session, request.ResourceIds)
		}
	}
}

func loadBalance(transactionServiceGroup string) string {
	addressList := getAddressList(transactionServiceGroup)
	if len(addressList) == 1 {
		return addressList[0]
	}
	return addressList[rand.Intn(len(addressList))]
}

func getAddressList(transactionServiceGroup string) []string {
	addressList := strings.Split(transactionServiceGroup, ",")
	return addressList
}

func (client *RpcRemoteClient) processMergedMessage() {
	ticker := time.NewTicker(5 * time.Millisecond)
	mergedMessage := protocol.MergedWarpMessage{
		Msgs:   make([]protocol.MessageTypeAware, 0),
		MsgIds: make([]int32, 0),
	}
	for {
		select {
		case rpcMessage := <-client.rpcMessageChannel:
			message := rpcMessage.Body.(protocol.MessageTypeAware)
			mergedMessage.Msgs = append(mergedMessage.Msgs, message)
			mergedMessage.MsgIds = append(mergedMessage.MsgIds, rpcMessage.ID)
			if len(mergedMessage.Msgs) == 20 {
				client.sendMergedMessage(mergedMessage)
				mergedMessage = protocol.MergedWarpMessage{
					Msgs:   make([]protocol.MessageTypeAware, 0),
					MsgIds: make([]int32, 0),
				}
			}
		case <-ticker.C:
			if len(mergedMessage.Msgs) > 0 {
				client.sendMergedMessage(mergedMessage)
				mergedMessage = protocol.MergedWarpMessage{
					Msgs:   make([]protocol.MessageTypeAware, 0),
					MsgIds: make([]int32, 0),
				}
			}
		}
	}
}

func (client *RpcRemoteClient) sendMergedMessage(mergedMessage protocol.MergedWarpMessage) {
	ss := clientSessionManager.AcquireGettySession()
	err := client.sendAsync(ss, mergedMessage)
	if err != nil {
		for _, id := range mergedMessage.MsgIds {
			resp, loaded := client.futures.Load(id)
			if loaded {
				response := resp.(*protocol.MessageFuture)
				response.Done <- true
				client.futures.Delete(id)
			}
		}
	}
}
