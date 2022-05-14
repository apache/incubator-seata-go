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
	"github.com/seata/seata-go/pkg/config"
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/rpc/processor"
	"github.com/seata/seata-go/pkg/utils/log"
)

var (
	clientHandler     *gettyClientHandler
	onceClientHandler = &sync.Once{}
)

type gettyClientHandler struct {
	conf           *config.ClientConfig
	idGenerator    *atomic.Uint32
	futures        *sync.Map
	mergeMsgMap    *sync.Map
	processorTable map[protocol.MessageType]processor.RemotingProcessor
}

func GetGettyClientHandlerInstance() *gettyClientHandler {
	if clientHandler == nil {
		onceClientHandler.Do(func() {
			clientHandler = &gettyClientHandler{
				conf:           config.GetClientConfig(),
				idGenerator:    &atomic.Uint32{},
				futures:        &sync.Map{},
				mergeMsgMap:    &sync.Map{},
				processorTable: make(map[protocol.MessageType]processor.RemotingProcessor, 0),
			}
		})
	}
	return clientHandler
}

// OnOpen ...
func (client *gettyClientHandler) OnOpen(session getty.Session) error {
	go func() {
		request := protocol.RegisterTMRequest{AbstractIdentifyRequest: protocol.AbstractIdentifyRequest{
			Version:                 client.conf.SeataVersion,
			ApplicationId:           client.conf.ApplicationID,
			TransactionServiceGroup: client.conf.TransactionServiceGroup,
		}}
		clientSessionManager.RegisterGettySession(session)
		err := GetGettyRemotingClient().SendAsyncRequest(request)
		//client.sendAsyncRequestWithResponse(session, request, RPC_REQUEST_TIMEOUT)
		if err != nil {
			log.Error("OnOpen error: {%#v}", err.Error())
			clientSessionManager.ReleaseGettySession(session)
			return
		}

		//todo
		//client.GettySessionOnOpenChannel <- session.RemoteAddr()
	}()

	return nil
}

// OnError ...
func (client *gettyClientHandler) OnError(session getty.Session, err error) {
	clientSessionManager.ReleaseGettySession(session)
}

// OnClose ...
func (client *gettyClientHandler) OnClose(session getty.Session) {
	clientSessionManager.ReleaseGettySession(session)
}

// OnMessage ...
func (client *gettyClientHandler) OnMessage(session getty.Session, pkg interface{}) {
	// TODO 需要把session里面的关键信息存储到context中，以方便在后面流程中获取使用。比如，XID等等
	ctx := context.Background()
	log.Debugf("received message: {%#v}", pkg)

	rpcMessage, ok := pkg.(protocol.RpcMessage)
	if !ok {
		log.Errorf("received message is not protocol.RpcMessage. pkg: %#v", pkg)
		return
	}

	if mm, ok := rpcMessage.Body.(protocol.MessageTypeAware); ok {
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

// OnCron ...
func (client *gettyClientHandler) OnCron(session getty.Session) {
	//GetGettyRemotingClient().SendAsyncRequest(protocol.HeartBeatMessagePing)
}

func (client *gettyClientHandler) RegisterProcessor(msgType protocol.MessageType, processor processor.RemotingProcessor) {
	if nil != processor {
		client.processorTable[msgType] = processor
	}
}
