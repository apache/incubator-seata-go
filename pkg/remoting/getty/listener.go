package getty

import (
	"context"
	"sync"
)

import (
	getty "github.com/apache/dubbo-getty"

	"go.uber.org/atomic"
)

import (
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/config"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/processor"
)

var (
	clientHandler     *gettyClientHandler
	onceClientHandler = &sync.Once{}
)

type gettyClientHandler struct {
	conf           *config.ClientConfig
	idGenerator    *atomic.Uint32
	msgFutures     *sync.Map
	mergeMsgMap    *sync.Map
	processorTable map[message.MessageType]processor.RemotingProcessor
}

func GetGettyClientHandlerInstance() *gettyClientHandler {
	if clientHandler == nil {
		onceClientHandler.Do(func() {
			clientHandler = &gettyClientHandler{
				conf:           config.GetDefaultClientConfig("seata-go"),
				idGenerator:    &atomic.Uint32{},
				msgFutures:     &sync.Map{},
				mergeMsgMap:    &sync.Map{},
				processorTable: make(map[message.MessageType]processor.RemotingProcessor, 0),
			}
		})
	}
	return clientHandler
}

func (client *gettyClientHandler) OnOpen(session getty.Session) error {
	sessionManager.RegisterGettySession(session)
	go func() {
		request := message.RegisterTMRequest{AbstractIdentifyRequest: message.AbstractIdentifyRequest{
			Version:                 client.conf.SeataVersion,
			ApplicationId:           client.conf.ApplicationID,
			TransactionServiceGroup: client.conf.TransactionServiceGroup,
		}}
		err := GetGettyRemotingClient().SendAsyncRequest(request)
		//client.sendAsyncRequestWithResponse(session, request, RPC_REQUEST_TIMEOUT)
		if err != nil {
			log.Error("OnOpen error: {%#v}", err.Error())
			sessionManager.ReleaseGettySession(session)
			return
		}

		//todo
		//client.GettySessionOnOpenChannel <- session.RemoteAddr()
	}()

	return nil
}

func (client *gettyClientHandler) OnError(session getty.Session, err error) {
	sessionManager.ReleaseGettySession(session)
}

func (client *gettyClientHandler) OnClose(session getty.Session) {
	sessionManager.ReleaseGettySession(session)
}

func (client *gettyClientHandler) OnMessage(session getty.Session, pkg interface{}) {
	// TODO 需要把session里面的关键信息存储到context中，以方便在后面流程中获取使用。比如，XID等等
	ctx := context.Background()
	log.Debugf("received message: {%#v}", pkg)

	rpcMessage, ok := pkg.(message.RpcMessage)
	if !ok {
		log.Errorf("received message is not protocol.RpcMessage. pkg: %#v", pkg)
		return
	}

	if mm, ok := rpcMessage.Body.(message.MessageTypeAware); ok {
		processor := client.processorTable[mm.GetTypeCode()]
		if processor != nil {
			processor.Process(ctx, rpcMessage)
		} else {
			log.Errorf("This message type [%v] has no processor.", mm.GetTypeCode())
		}
	} else {
		log.Errorf("This rpcMessage body[%v] is not MessageTypeAware type.", rpcMessage.Body)
	}
}

func (client *gettyClientHandler) OnCron(session getty.Session) {
	// todo 发送心跳消息
}

func (client *gettyClientHandler) RegisterProcessor(msgType message.MessageType, processor processor.RemotingProcessor) {
	if nil != processor {
		client.processorTable[msgType] = processor
	}
}
