package getty

import (
	"sync"
	"time"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/protocol/codec"
)

var (
	gettyRemotingClient     *GettyRemotingClient
	onceGettyRemotingClient = &sync.Once{}
)

type GettyRemotingClient struct {
	//conf                         *config.ClientConfig
	idGenerator *atomic.Uint32
	//futures                      *sync.Map
	//mergeMsgMap                  *sync.Map
	//rpcMessageChannel            chan protocol.RpcMessage
	//BranchCommitRequestChannel   chan RpcRMMessage
	//BranchRollbackRequestChannel chan RpcRMMessage
	//GettySessionOnOpenChannel    chan string
}

func GetGettyRemotingClient() *GettyRemotingClient {
	if gettyRemotingClient == nil {
		onceGettyRemotingClient.Do(func() {
			gettyRemotingClient = &GettyRemotingClient{
				idGenerator: &atomic.Uint32{},
			}
		})
	}
	return gettyRemotingClient
}

func (client *GettyRemotingClient) SendAsyncRequest(msg interface{}) error {
	var msgType protocol.RequestType
	if _, ok := msg.(protocol.HeartBeatMessage); ok {
		msgType = protocol.MSGTypeHeartbeatRequest
	} else {
		msgType = protocol.MSGTypeRequestOneway
	}
	rpcMessage := protocol.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       msgType,
		Codec:      codec.SEATA,
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendASync(rpcMessage)
}

func (client *GettyRemotingClient) SendSyncRequest(msg interface{}) (interface{}, error) {
	rpcMessage := protocol.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       protocol.MSGTypeRequestSync,
		Codec:      codec.SEATA,
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendSync(rpcMessage)
}

func (client *GettyRemotingClient) SendSyncRequestWithTimeout(msg interface{}, timeout time.Duration) (interface{}, error) {
	rpcMessage := protocol.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       protocol.MSGTypeRequestSync,
		Codec:      codec.SEATA,
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendSyncWithTimeout(rpcMessage, timeout)
}

//
//func (client *GettyRemotingClient) RegisterResource(serverAddress string, request protocol.RegisterRMRequest) {
//	session := clientSessionManager.AcquireGettySessionByServerAddress(serverAddress)
//	if session != nil {
//		err := client.sendAsyncRequestWithoutResponse(session, request)
//		if err != nil {
//			log.Errorf("register resource failed, session:{},resourceID:{}", session, request.ResourceIds)
//		}
//	}
//}
//
//func loadBalance(transactionServiceGroup string) string {
//	addressList := getAddressList(transactionServiceGroup)
//	if len(addressList) == 1 {
//		return addressList[0]
//	}
//	return addressList[rand.Intn(len(addressList))]
//}
//
//func getAddressList(transactionServiceGroup string) []string {
//	addressList := strings.Split(transactionServiceGroup, ",")
//	return addressList
//}
//
//func (client *GettyRemotingClient) processMergedMessage() {
//	ticker := time.NewTicker(5 * time.Millisecond)
//	mergedMessage := protocol.MergedWarpMessage{
//		Msgs:   make([]protocol.MessageTypeAware, 0),
//		MsgIds: make([]int32, 0),
//	}
//	for {
//		select {
//		case rpcMessage := <-client.rpcMessageChannel:
//			message := rpcMessage.Body.(protocol.MessageTypeAware)
//			mergedMessage.Msgs = append(mergedMessage.Msgs, message)
//			mergedMessage.MsgIds = append(mergedMessage.MsgIds, rpcMessage.ID)
//			if len(mergedMessage.Msgs) == 20 {
//				client.sendMergedMessage(mergedMessage)
//				mergedMessage = protocol.MergedWarpMessage{
//					Msgs:   make([]protocol.MessageTypeAware, 0),
//					MsgIds: make([]int32, 0),
//				}
//			}
//		case <-ticker.C:
//			if len(mergedMessage.Msgs) > 0 {
//				client.sendMergedMessage(mergedMessage)
//				mergedMessage = protocol.MergedWarpMessage{
//					Msgs:   make([]protocol.MessageTypeAware, 0),
//					MsgIds: make([]int32, 0),
//				}
//			}
//		}
//	}
//}
//
//func (client *GettyRemotingClient) sendMergedMessage(mergedMessage protocol.MergedWarpMessage) {
//	ss := clientSessionManager.AcquireGettySession()
//	err := client.sendAsync(ss, mergedMessage)
//	if err != nil {
//		for _, id := range mergedMessage.MsgIds {
//			resp, loaded := client.futures.Load(id)
//			if loaded {
//				response := resp.(*protocol.MessageFuture)
//				response.Done <- true
//				client.futures.Delete(id)
//			}
//		}
//	}
//}
