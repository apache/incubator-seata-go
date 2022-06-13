package getty

import (
	"sync"
	"time"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/seata/seata-go/pkg/protocol/codec"
	"github.com/seata/seata-go/pkg/protocol/message"
)

var (
	gettyRemotingClient     *GettyRemotingClient
	onceGettyRemotingClient = &sync.Once{}
)

type GettyRemotingClient struct {
	idGenerator *atomic.Uint32
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
	var msgType message.GettyRequestType
	if _, ok := msg.(message.HeartBeatMessage); ok {
		msgType = message.GettyRequestType_HeartbeatRequest
	} else {
		msgType = message.GettyRequestType_RequestOneway
	}
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       msgType,
		Codec:      byte(codec.CodeTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendASync(rpcMessage)
}

func (client *GettyRemotingClient) SendAsyncResponse(msg interface{}) error {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.GettyRequestType_Response,
		Codec:      byte(codec.CodeTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendASync(rpcMessage)
}

func (client *GettyRemotingClient) SendSyncRequest(msg interface{}) (interface{}, error) {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.GettyRequestType_RequestSync,
		Codec:      byte(codec.CodeTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendSync(rpcMessage)
}

func (client *GettyRemotingClient) SendSyncRequestWithTimeout(msg interface{}, timeout time.Duration) (interface{}, error) {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.GettyRequestType_RequestSync,
		Codec:      byte(codec.CodeTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendSyncWithTimeout(rpcMessage, timeout)
}
