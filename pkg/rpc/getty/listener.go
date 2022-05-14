package getty

import (
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
				conf:        config.GetClientConfig(),
				idGenerator: &atomic.Uint32{},
				futures:     &sync.Map{},
				mergeMsgMap: &sync.Map{},
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

	if rpcMessage.Type == protocol.MSGTypeRequestSync ||
		rpcMessage.Type == protocol.MSGTypeRequestOneway {
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
func (client *gettyClientHandler) OnCron(session getty.Session) {
	//GetGettyRemotingClient().SendAsyncRequest(protocol.HeartBeatMessagePing)
}

func (client *gettyClientHandler) onMessage(rpcMessage protocol.RpcMessage, serverAddress string) {
	msg := rpcMessage.Body.(protocol.MessageTypeAware)
	log.Debugf("onMessage: %#v", msg)
	// todo
	//switch msg.GetTypeCode() {
	//case protocol.TypeBranchCommit:
	//	client.BranchCommitRequestChannel <- RpcRMMessage{
	//		RpcMessage:    rpcMessage,
	//		ServerAddress: serverAddress,
	//	}
	//case protocol.TypeBranchRollback:
	//	client.BranchRollbackRequestChannel <- RpcRMMessage{
	//		RpcMessage:    rpcMessage,
	//		ServerAddress: serverAddress,
	//	}
	//case protocol.TypeRmDeleteUndolog:
	//	break
	//default:
	//	break
	//}
}

func (client *gettyClientHandler) RegisterProcessor(msgType protocol.MessageType, processor processor.RemotingProcessor) {
	if nil != processor {
		client.processorTable[msgType] = processor
	}
}
