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

func (client *GettyRemotingClient) SendAsyncResponse(msg interface{}) error {
	rpcMessage := protocol.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       protocol.MSGTypeResponse,
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
