package rpc_client

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	getty2 "github.com/transaction-wg/seata-golang/pkg/base/getty"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal/codec"
	"github.com/transaction-wg/seata-golang/pkg/client/config"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/runtime"
)

const (
	RPC_REQUEST_TIMEOUT = 30 * time.Second
)

var rpcRemoteClient *RpcRemoteClient

func InitRpcRemoteClient() *RpcRemoteClient {
	rpcRemoteClient = &RpcRemoteClient{
		conf:                         config.GetClientConfig(),
		idGenerator:                  atomic.Uint32{},
		futures:                      &sync.Map{},
		mergeMsgMap:                  &sync.Map{},
		rpcMessageChannel:            make(chan protocal.RpcMessage, 100),
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
	conf                         config.ClientConfig
	idGenerator                  atomic.Uint32
	futures                      *sync.Map
	mergeMsgMap                  *sync.Map
	rpcMessageChannel            chan protocal.RpcMessage
	BranchCommitRequestChannel   chan RpcRMMessage
	BranchRollbackRequestChannel chan RpcRMMessage
	GettySessionOnOpenChannel    chan string
}

// OnOpen ...
func (client *RpcRemoteClient) OnOpen(session getty.Session) error {
	go func() {
		request := protocal.RegisterTMRequest{AbstractIdentifyRequest: protocal.AbstractIdentifyRequest{
			Version:                 client.conf.SeataVersion,
			ApplicationID:           client.conf.ApplicationID,
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
	log.Info("received message:{%v}", pkg)
	rpcMessage, ok := pkg.(protocal.RpcMessage)
	if ok {
		heartBeat, isHeartBeat := rpcMessage.Body.(protocal.HeartBeatMessage)
		if isHeartBeat && heartBeat == protocal.HeartBeatMessagePong {
			log.Debugf("received PONG from %s", session.RemoteAddr())
			return
		}
	}

	if rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST ||
		rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST_ONEWAY {
		log.Debugf("msgID:%s, body:%v", rpcMessage.ID, rpcMessage.Body)

		client.onMessage(rpcMessage, session.RemoteAddr())
	} else {
		mergedResult, isMergedResult := rpcMessage.Body.(protocal.MergeResultMessage)
		if isMergedResult {
			mm, loaded := client.mergeMsgMap.Load(rpcMessage.ID)
			if loaded {
				mergedMessage := mm.(protocal.MergedWarpMessage)
				log.Infof("rpcMessageID: %d,rpcMessage :%v,result :%v", rpcMessage.ID, mergedMessage, mergedResult)
				for i := 0; i < len(mergedMessage.Msgs); i++ {
					msgID := mergedMessage.MsgIDs[i]
					resp, loaded := client.futures.Load(msgID)
					if loaded {
						response := resp.(*getty2.MessageFuture)
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
				response := resp.(*getty2.MessageFuture)
				response.Response = rpcMessage.Body
				response.Done <- true
				client.futures.Delete(rpcMessage.ID)
			}
		}
	}
}

// OnCron ...
func (client *RpcRemoteClient) OnCron(session getty.Session) {
	client.defaultSendRequest(session, protocal.HeartBeatMessagePing)
}

func (client *RpcRemoteClient) onMessage(rpcMessage protocal.RpcMessage, serverAddress string) {
	msg := rpcMessage.Body.(protocal.MessageTypeAware)
	log.Infof("onMessage: %v", msg)
	switch msg.GetTypeCode() {
	case protocal.TypeBranchCommit:
		client.BranchCommitRequestChannel <- RpcRMMessage{
			RpcMessage:    rpcMessage,
			ServerAddress: serverAddress,
		}
	case protocal.TypeBranchRollback:
		client.BranchRollbackRequestChannel <- RpcRMMessage{
			RpcMessage:    rpcMessage,
			ServerAddress: serverAddress,
		}
	case protocal.TypeRmDeleteUndolog:
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

func (client *RpcRemoteClient) SendResponse(request protocal.RpcMessage, serverAddress string, msg interface{}) {
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
	rpcMessage := protocal.RpcMessage{
		ID:          int32(client.idGenerator.Inc()),
		MessageType: protocal.MSGTYPE_RESQUEST_ONEWAY,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	resp := getty2.NewMessageFuture(rpcMessage)
	client.futures.Store(rpcMessage.ID, resp)
	//config timeout
	_, _, err = session.WritePkg(rpcMessage, time.Duration(0))
	if err != nil {
		client.futures.Delete(rpcMessage.ID)
	}
	log.Infof("send message : %v,session:%s", rpcMessage, session.Stat())

	if timeout > time.Duration(0) {
		select {
		case <-getty.GetTimeWheel().After(timeout):
			client.futures.Delete(rpcMessage.ID)
			return nil, errors.Errorf("wait response timeout,ip:%s,request:%v", session.RemoteAddr(), rpcMessage)
		case <-resp.Done:
			err = resp.Err
		}
		return resp.Response, err
	}
	return nil, err
}

func (client *RpcRemoteClient) sendAsyncRequest2(msg interface{}, timeout time.Duration) (interface{}, error) {
	var err error
	rpcMessage := protocal.RpcMessage{
		ID:          int32(client.idGenerator.Inc()),
		MessageType: protocal.MSGTYPE_RESQUEST,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	resp := getty2.NewMessageFuture(rpcMessage)
	client.futures.Store(rpcMessage.ID, resp)

	client.rpcMessageChannel <- rpcMessage

	log.Infof("send message : %v", rpcMessage)

	if timeout > time.Duration(0) {
		select {
		case <-getty.GetTimeWheel().After(timeout):
			client.futures.Delete(rpcMessage.ID)
			return nil, errors.Errorf("wait response timeout, request:%v", rpcMessage)
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
	rpcMessage := protocal.RpcMessage{
		ID:          int32(client.idGenerator.Inc()),
		MessageType: protocal.MSGTYPE_RESQUEST_ONEWAY,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	log.Infof("store message,id %d: %v", rpcMessage.ID, msg)
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
	rpcMessage := protocal.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Codec:      codec.SEATA,
		Compressor: 0,
		Body:       msg,
	}
	_, ok := msg.(protocal.HeartBeatMessage)
	if ok {
		rpcMessage.MessageType = protocal.MSGTYPE_HEARTBEAT_REQUEST
	} else {
		rpcMessage.MessageType = protocal.MSGTYPE_RESQUEST
	}
	pkgLen, sendLen, err := session.WritePkg(rpcMessage, client.conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		runtime.GoWithRecover(func() {
			session.Close()
		}, nil)
	}
}

func (client *RpcRemoteClient) defaultSendResponse(request protocal.RpcMessage, session getty.Session, msg interface{}) {
	resp := protocal.RpcMessage{
		ID:         request.ID,
		Codec:      request.Codec,
		Compressor: request.Compressor,
		Body:       msg,
	}
	_, ok := msg.(protocal.HeartBeatMessage)
	if ok {
		resp.MessageType = protocal.MSGTYPE_HEARTBEAT_RESPONSE
	} else {
		resp.MessageType = protocal.MSGTYPE_RESPONSE
	}

	pkgLen, sendLen, err := session.WritePkg(resp, time.Duration(0))
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		runtime.GoWithRecover(func() {
			session.Close()
		}, nil)
	}
}

func (client *RpcRemoteClient) RegisterResource(serverAddress string, request protocal.RegisterRMRequest) {
	session := clientSessionManager.AcquireGettySessionByServerAddress(serverAddress)
	if session != nil {
		err := client.sendAsyncRequestWithoutResponse(session, request)
		if err != nil {
			log.Errorf("register resource failed, session:{},resourceID:{}", session, request.ResourceIDs)
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
	mergedMessage := protocal.MergedWarpMessage{
		Msgs:   make([]protocal.MessageTypeAware, 0),
		MsgIDs: make([]int32, 0),
	}
	for {
		select {
		case rpcMessage := <-client.rpcMessageChannel:
			message := rpcMessage.Body.(protocal.MessageTypeAware)
			mergedMessage.Msgs = append(mergedMessage.Msgs, message)
			mergedMessage.MsgIDs = append(mergedMessage.MsgIDs, rpcMessage.ID)
			if len(mergedMessage.Msgs) == 20 {
				client.sendMergedMessage(mergedMessage)
				mergedMessage = protocal.MergedWarpMessage{
					Msgs:   make([]protocal.MessageTypeAware, 0),
					MsgIDs: make([]int32, 0),
				}
			}
		case <-ticker.C:
			if len(mergedMessage.Msgs) > 0 {
				client.sendMergedMessage(mergedMessage)
				mergedMessage = protocal.MergedWarpMessage{
					Msgs:   make([]protocal.MessageTypeAware, 0),
					MsgIDs: make([]int32, 0),
				}
			}
		}
	}
}

func (client *RpcRemoteClient) sendMergedMessage(mergedMessage protocal.MergedWarpMessage) {
	ss := clientSessionManager.AcquireGettySession()
	err := client.sendAsync(ss, mergedMessage)
	if err != nil {
		for _, id := range mergedMessage.MsgIDs {
			resp, loaded := client.futures.Load(id)
			if loaded {
				response := resp.(*getty2.MessageFuture)
				response.Done <- true
				client.futures.Delete(id)
			}
		}
	}
}
