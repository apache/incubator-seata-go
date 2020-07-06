package getty

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/getty"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	getty2 "github.com/dk-lockdown/seata-golang/base/getty"
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"github.com/dk-lockdown/seata-golang/base/protocal/codec"
	"github.com/dk-lockdown/seata-golang/client/config"
	"github.com/dk-lockdown/seata-golang/client/model"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
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
		BranchRollbackRequestChannel: make(chan model.RpcRMMessage),
		BranchCommitRequestChannel:   make(chan model.RpcRMMessage),
		GettySessionOnOpenChannel:    make(chan string),
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
	BranchCommitRequestChannel   chan model.RpcRMMessage
	BranchRollbackRequestChannel chan model.RpcRMMessage
	GettySessionOnOpenChannel    chan string
}

// OnOpen ...
func (client *RpcRemoteClient) OnOpen(session getty.Session) error {
	go func() {
		request := protocal.RegisterTMRequest{AbstractIdentifyRequest: protocal.AbstractIdentifyRequest{
			ApplicationId:           client.conf.ApplicationId,
			TransactionServiceGroup: client.conf.TransactionServiceGroup,
		}}
		_, err := client.sendAsyncRequestWithResponse("", session, request, RPC_REQUEST_TIMEOUT)
		if err == nil {
			clientSessionManager.RegisterGettySession(session, session.RemoteAddr())
			client.GettySessionOnOpenChannel <- session.RemoteAddr()
		}
	}()

	return nil
}

// OnError ...
func (client *RpcRemoteClient) OnError(session getty.Session, err error) {
	clientSessionManager.ReleaseGettySession(session, session.RemoteAddr())
}

// OnClose ...
func (client *RpcRemoteClient) OnClose(session getty.Session) {
	clientSessionManager.ReleaseGettySession(session, session.RemoteAddr())
}

// OnMessage ...
func (client *RpcRemoteClient) OnMessage(session getty.Session, pkg interface{}) {
	logging.Logger.Info("received message:{%v}", pkg)
	rpcMessage, ok := pkg.(protocal.RpcMessage)
	if ok {
		heartBeat, isHeartBeat := rpcMessage.Body.(protocal.HeartBeatMessage)
		if isHeartBeat && heartBeat == protocal.HeartBeatMessagePong {
			logging.Logger.Debugf("received PONG from %s", session.RemoteAddr())
		}
	}

	if rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST ||
		rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST_ONEWAY {
		logging.Logger.Debugf("msgId:%s, body:%v", rpcMessage.Id, rpcMessage.Body)

		client.onMessage(rpcMessage, session.RemoteAddr())
	} else {
		resp, loaded := client.futures.Load(rpcMessage.Id)
		if loaded {
			response := resp.(*getty2.MessageFuture)
			response.Response = rpcMessage.Body
			response.Done <- true
			client.futures.Delete(rpcMessage.Id)
		}
	}
}

// OnCron ...
func (client *RpcRemoteClient) OnCron(session getty.Session) {
	client.defaultSendRequest(session, protocal.HeartBeatMessagePing)
}

func (client *RpcRemoteClient) onMessage(rpcMessage protocal.RpcMessage, serverAddress string) {
	msg := rpcMessage.Body.(protocal.MessageTypeAware)
	logging.Logger.Infof("onMessage: %v", msg)
	switch msg.GetTypeCode() {
	case protocal.TypeBranchCommit:
		client.BranchCommitRequestChannel <- model.RpcRMMessage{
			RpcMessage:    rpcMessage,
			ServerAddress: serverAddress,
		}
	case protocal.TypeBranchRollback:
		client.BranchRollbackRequestChannel <- model.RpcRMMessage{
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
	return client.SendMsgWithResponseAndTimeout(msg, RPC_REQUEST_TIMEOUT)
}

func (client *RpcRemoteClient) SendMsgWithResponseAndTimeout(msg interface{}, timeout time.Duration) (interface{}, error) {
	validAddress := loadBalance(client.conf.TransactionServiceGroup)
	ss := clientSessionManager.AcquireGettySession(validAddress)
	return client.sendAsyncRequestWithResponse(validAddress, ss, msg, timeout)
}

func (client *RpcRemoteClient) SendMsgByServerAddressWithResponseAndTimeout(serverAddress string, msg interface{}, timeout time.Duration) (interface{}, error) {
	return client.sendAsyncRequestWithResponse(serverAddress, clientSessionManager.AcquireGettySession(serverAddress), msg, timeout)
}

func (client *RpcRemoteClient) SendResponse(request protocal.RpcMessage, serverAddress string, msg interface{}) {
	client.defaultSendResponse(request, clientSessionManager.AcquireGettySession(serverAddress), msg)
}

func (client *RpcRemoteClient) sendAsyncRequestWithResponse(address string, session getty.Session, msg interface{}, timeout time.Duration) (interface{}, error) {
	if timeout <= time.Duration(0) {
		return nil, errors.New("timeout should more than 0ms")
	}
	return client.sendAsyncRequest(address, session, msg, timeout)
}

func (client *RpcRemoteClient) sendAsyncRequestWithoutResponse(session getty.Session, msg interface{}) error {
	_, err := client.sendAsyncRequest("", session, msg, time.Duration(0))
	return err
}

func (client *RpcRemoteClient) sendAsyncRequest(address string, session getty.Session, msg interface{}, timeout time.Duration) (interface{}, error) {
	var err error
	if session == nil {
		logging.Logger.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	rpcMessage := protocal.RpcMessage{
		Id:          int32(client.idGenerator.Inc()),
		MessageType: protocal.MSGTYPE_RESQUEST_ONEWAY,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	resp := getty2.NewMessageFuture(rpcMessage)
	client.futures.Store(rpcMessage.Id, resp)
	//config timeout
	err = session.WritePkg(rpcMessage, time.Duration(0))
	if err != nil {
		client.futures.Delete(rpcMessage.Id)
	}
	logging.Logger.Infof("send message : %v,session:%s", rpcMessage, session.Stat())

	if timeout > time.Duration(0) {
		select {
		case <-getty.GetTimeWheel().After(timeout):
			client.futures.Delete(rpcMessage.Id)
			return nil, errors.Errorf("wait response timeout,ip:%s,request:%v", address, rpcMessage)
		case <-resp.Done:
			err = resp.Err
		}
		return resp.Response, err
	}
	return nil, err
}

func (client *RpcRemoteClient) RegisterResource(serverAddress string, request protocal.RegisterRMRequest) {
	session, ok := sessions.Load(serverAddress)
	if ok {
		rmSession := session.(getty.Session)
		err := client.sendAsyncRequestWithoutResponse(rmSession, request)
		if err != nil {
			logging.Logger.Errorf("register resource failed, session:{},resourceId:{}", rmSession, request.ResourceIds)
		}
	}
}

func (client *RpcRemoteClient) defaultSendRequest(session getty.Session, msg interface{}) {
	rpcMessage := protocal.RpcMessage{
		Id:         int32(client.idGenerator.Inc()),
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
	session.WritePkg(rpcMessage, client.conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
}

func (client *RpcRemoteClient) defaultSendResponse(request protocal.RpcMessage, session getty.Session, msg interface{}) {
	resp := protocal.RpcMessage{
		Id:         request.Id,
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

	session.WritePkg(resp, time.Duration(0))
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
