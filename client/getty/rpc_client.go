package getty

import (
	getty2 "github.com/dk-lockdown/seata-golang/base/getty"
	"github.com/dk-lockdown/seata-golang/base/protocal"
	"github.com/dk-lockdown/seata-golang/base/protocal/codec"
	"github.com/dk-lockdown/seata-golang/client/config"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dubbogo/getty"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	RPC_REQUEST_TIMEOUT = 30 * time.Second
)

var RpcClient *Client

func InitRpcClient() {
	RpcClient = &Client{
		conf:        config.GetClientConfig(),
		idGenerator: atomic.Uint32{},
		futures:     &sync.Map{},
	}
}

type Client struct {
	conf config.ClientConfig
	idGenerator atomic.Uint32
	futures *sync.Map
}

// OnOpen ...
func (client *Client) OnOpen(session getty.Session) error {
	request := protocal.RegisterTMRequest{AbstractIdentifyRequest:protocal.AbstractIdentifyRequest{
		ApplicationId:           client.conf.ApplicationId,
		TransactionServiceGroup: client.conf.TransactionServiceGroup,
	}}
	_, err := client.sendAsyncRequestWithResponse("",session,request,RPC_REQUEST_TIMEOUT)
	if err != nil {
		clientSessionManager.RegisterGettySession(session, session.RemoteAddr())
	}
	return nil
}

// OnError ...
func (client *Client) OnError(session getty.Session, err error) {
	clientSessionManager.ReleaseGettySession(session,session.RemoteAddr())
}

// OnClose ...
func (client *Client) OnClose(session getty.Session) {
	clientSessionManager.ReleaseGettySession(session,session.RemoteAddr())
}

// OnMessage ...
func (client *Client) OnMessage(session getty.Session, pkg interface{}) {
	logging.Logger.Info("received message:{%v}", pkg)
	rpcMessage,ok := pkg.(protocal.RpcMessage)
	if ok {
		heartBeat,isHeartBeat := rpcMessage.Body.(protocal.HeartBeatMessage)
		if isHeartBeat && heartBeat == protocal.HeartBeatMessagePong {
			logging.Logger.Debugf("received PONG from %s", session.RemoteAddr())
		}
	}

	if rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST ||
		rpcMessage.MessageType == protocal.MSGTYPE_RESQUEST_ONEWAY {
		logging.Logger.Debugf("msgId:%s, body:%v", rpcMessage.Id, rpcMessage.Body)
		// todo transaction branch things
	} else {
		resp,loaded := client.futures.Load(rpcMessage.Id)
		if loaded {
			response := resp.(*getty2.MessageFuture)
			response.Response = rpcMessage.Body
			response.Done <- true
			client.futures.Delete(rpcMessage.Id)
		}
	}
}

// OnCron ...
func (client *Client) OnCron(session getty.Session) {
	client.defaultSendRequest(session,protocal.HeartBeatMessagePing)
}

func (client *Client) SendMsgWithResponse(msg interface{}) (interface{},error) {
	return client.SendMsgWithResponseAndTimeout(msg, RPC_REQUEST_TIMEOUT)
}

func (client *Client) SendMsgWithResponseAndTimeout(msg interface{}, timeout time.Duration) (interface{},error) {
	validAddress := loadBalance(client.conf.TransactionServiceGroup)
	ss := clientSessionManager.AcquireGettySession(validAddress)
	return client.sendAsyncRequestWithResponse(validAddress,ss,msg,timeout)
}


func (client *Client) SendMsgByServerAddressWithResponseAndTimeout(serverAddress string, msg interface{}, timeout time.Duration) (interface{},error) {
	return client.sendAsyncRequestWithResponse(serverAddress,clientSessionManager.AcquireGettySession(serverAddress),msg,timeout)
}

func (client *Client) SendResponse(request protocal.RpcMessage, serverAddress string, msg interface{}) {
	client.defaultSendResponse(request,clientSessionManager.AcquireGettySession(serverAddress),msg)
}

func (client *Client) sendAsyncRequestWithResponse(address string,session getty.Session,msg interface{},timeout time.Duration) (interface{},error) {
	if timeout <= time.Duration(0) {
		return nil,errors.New("timeout should more than 0ms")
	}
	return client.sendAsyncRequest(address,session,msg,timeout)
}

func (client *Client) sendAsyncRequestWithoutResponse(session getty.Session,msg interface{}) error {
	_,err := client.sendAsyncRequest("",session,msg,time.Duration(0))
	return err
}

func (client *Client) sendAsyncRequest(address string,session getty.Session,msg interface{},timeout time.Duration) (interface{},error) {
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
	err = session.WritePkg(rpcMessage, client.conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
	if err != nil {
		client.futures.Delete(rpcMessage.Id)
	}

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
	return nil,err
}

func (client *Client) defaultSendRequest(session getty.Session, msg interface{}) {
	rpcMessage := protocal.RpcMessage{
		Id:          int32(client.idGenerator.Inc()),
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	_,ok := msg.(protocal.HeartBeatMessage)
	if ok {
		rpcMessage.MessageType = protocal.MSGTYPE_HEARTBEAT_REQUEST
	} else {
		rpcMessage.MessageType = protocal.MSGTYPE_RESQUEST
	}
	session.WritePkg(rpcMessage, client.conf.GettyConfig.GettySessionParam.TcpWriteTimeout)
}

func (client *Client) defaultSendResponse(request protocal.RpcMessage, session getty.Session, msg interface{}) {
	resp := protocal.RpcMessage{
		Id:          request.Id,
		Codec:       request.Codec,
		Compressor:  request.Compressor,
		Body:        msg,
	}
	_,ok := msg.(protocal.HeartBeatMessage)
	if ok {
		resp.MessageType = protocal.MSGTYPE_HEARTBEAT_RESPONSE
	} else {
		resp.MessageType = protocal.MSGTYPE_RESPONSE
	}
	session.WritePkg(resp,time.Duration(0))
}


func loadBalance(transactionServiceGroup string) string {
	addressList := strings.Split(transactionServiceGroup,",")
	if len(addressList) == 1 {
		return addressList[0]
	}
	return addressList[rand.Intn(len(addressList))]
}