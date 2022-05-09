package rpc11

import (
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/protocol/codec"
	log "github.com/seata/seata-go/pkg/util/log"
)

type ClientEventHandler struct {
}

func NewClientEventHandler() *ClientEventHandler {
	return &ClientEventHandler{}
}

func (h *ClientEventHandler) OnOpen(session getty.Session) error {
	log.Infof("OnOpen session{%s} open", session.Stat())
	rpcClient.RegisterGettySession(session)
	return nil
}

func (h *ClientEventHandler) OnError(session getty.Session, err error) {
	log.Infof("OnError session{%s} got error{%v}, will be closed.", session.Stat(), err)
}

func (h *ClientEventHandler) OnClose(session getty.Session) {
	log.Infof("OnClose session{%s} is closing......", session.Stat())
}

func (h *ClientEventHandler) OnMessage(session getty.Session, pkg interface{}) {
	s, ok := pkg.(string)
	if !ok {
		log.Infof("illegal packge{%#v}", pkg)
		return
	}

	log.Infof("OnMessage: %s", s)
}

func (h *ClientEventHandler) OnCron(session getty.Session) {
	active := session.GetActive()
	if 20*time.Second.Nanoseconds() < time.Since(active).Nanoseconds() {
		log.Infof("OnCorn session{%s} timeout{%s}", session.Stat(), time.Since(active).String())
		session.Close()
	}
}

func (client *ClientEventHandler) sendMergedMessage(mergedMessage protocol.MergedWarpMessage) {
	ss := rpcClient.AcquireGettySession()
	err := client.sendAsync(ss, mergedMessage)
	if err != nil {
		log.Errorf("error sendMergedMessage")
	}
}

func (client *ClientEventHandler) sendAsync(session getty.Session, msg interface{}) error {
	var err error
	if session == nil || session.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	idGenerator := atomic.Uint32{}
	rpcMessage := protocol.RpcMessage{
		ID:          int32(idGenerator.Inc()),
		MessageType: protocol.MSGTypeRequestOneway,
		Codec:       codec.SEATA,
		Compressor:  0,
		Body:        msg,
	}
	log.Infof("store message, id %d : %#v", rpcMessage.ID, msg)
	//client.mergeMsgMap.Store(rpcMessage.ID, msg)
	//config timeout
	pkgLen, sendLen, err := session.WritePkg(rpcMessage, time.Duration(0))
	if err != nil || (pkgLen != 0 && pkgLen != sendLen) {
		log.Warnf("start to close the session because %d of %d bytes data is sent success. err:%+v", sendLen, pkgLen, err)
		//runtime.GoWithRecover(func() {
		//	session.Close()
		//}, nil)
		return errors.Wrap(err, "pkg not send completely!")
	}
	return nil
}
