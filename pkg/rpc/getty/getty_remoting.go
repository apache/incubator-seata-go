package getty

import (
	"sync"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"

	gxtime "github.com/dubbogo/gost/time"

	"github.com/pkg/errors"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/utils/log"
)

const (
	RPC_REQUEST_TIMEOUT = 30 * time.Second
)

var (
	gettyRemoting     *GettyRemoting
	onceGettyRemoting = &sync.Once{}
)

type GettyRemoting struct {
	futures     *sync.Map
	mergeMsgMap *sync.Map
}

func GetGettyRemotingInstance() *GettyRemoting {
	if gettyRemoting == nil {
		onceGettyRemoting.Do(func() {
			gettyRemoting = &GettyRemoting{
				futures:     &sync.Map{},
				mergeMsgMap: &sync.Map{},
			}
		})
	}
	return gettyRemoting
}

func (client *GettyRemoting) SendSync(msg protocol.RpcMessage) (interface{}, error) {
	ss := clientSessionManager.AcquireGettySession()
	return client.sendAsync(ss, msg, RPC_REQUEST_TIMEOUT)
}

func (client *GettyRemoting) SendSyncWithTimeout(msg protocol.RpcMessage, timeout time.Duration) (interface{}, error) {
	ss := clientSessionManager.AcquireGettySession()
	return client.sendAsync(ss, msg, timeout)
}

func (client *GettyRemoting) SendASync(msg protocol.RpcMessage) error {
	ss := clientSessionManager.AcquireGettySession()
	_, err := client.sendAsync(ss, msg, 0*time.Second)
	return err
}

func (client *GettyRemoting) sendAsync(session getty.Session, msg protocol.RpcMessage, timeout time.Duration) (interface{}, error) {
	var err error
	if session == nil || session.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	resp := protocol.NewMessageFuture(msg)
	client.futures.Store(msg.ID, resp)
	_, _, err = session.WritePkg(msg, time.Duration(0))
	if err != nil {
		client.futures.Delete(msg.ID)
		log.Errorf("send message: %#v, session: %s", msg, session.Stat())
		return nil, err
	}

	log.Debugf("send message: %#v, session: %s", msg, session.Stat())

	if timeout > time.Duration(0) {
		select {
		case <-gxtime.GetDefaultTimerWheel().After(timeout):
			client.futures.Delete(msg.ID)
			if session != nil {
				return nil, errors.Errorf("wait response timeout, ip: %s, request: %#v", session.RemoteAddr(), msg)
			} else {
				return nil, errors.Errorf("wait response timeout and session is nil, request: %#v", msg)
			}
		case <-resp.Done:
			err = resp.Err
			return resp.Response, err
		}
	}

	return nil, err
}

func (client *GettyRemoting) GetMessageFuture(msgID int32) *protocol.MessageFuture {
	if msg, ok := client.futures.Load(msgID); ok {
		return msg.(*protocol.MessageFuture)
	}
	return nil
}

func (client *GettyRemoting) RemoveMessageFuture(msgID int32) {
	client.futures.Delete(msgID)
}

func (client *GettyRemoting) RemoveMergedMessageFuture(msgID int32) {
	client.mergeMsgMap.Delete(msgID)
}

func (client *GettyRemoting) GetMergedMessage(msgID int32) *protocol.MergedWarpMessage {
	if msg, ok := client.mergeMsgMap.Load(msgID); ok {
		return msg.(*protocol.MergedWarpMessage)
	}
	return nil
}

func (client *GettyRemoting) NotifytRpcMessageResponse(rpcMessage protocol.RpcMessage) {
	messageFuture := client.GetMessageFuture(rpcMessage.ID)
	if messageFuture != nil {
		messageFuture.Response = rpcMessage.Body
		// todo messageFuture.Err怎么配置呢？
		//messageFuture.Err = rpcMessage.Err
		messageFuture.Done <- true
		//client.futures.Delete(rpcMessage.ID)
	} else {
		log.Infof("msg: {} is not found in futures.", rpcMessage.ID)
	}
}
