package rocketmq

import (
	"context"
	"errors"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
	"strings"
)

type TCCRocketMQImpl struct {
	producer *MQProducer
}

func NewTCCRocketMQImpl() *TCCRocketMQImpl {
	return &TCCRocketMQImpl{}
}

func (t *TCCRocketMQImpl) SetProducer(producer *MQProducer) {
	t.producer = producer
}

func (t *TCCRocketMQImpl) Prepare(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	businessActionCtx := tm.GetBusinessActionContext(ctx)
	if businessActionCtx == nil {
		return nil, errors.New("business action context not found")
	}

	result, err := t.producer.SendMsgInTransaction(msg, businessActionCtx.Xid, businessActionCtx.BranchId)
	if err != nil {
		log.Errorf("Failed to send transaction message: %v", err)
		return nil, err
	}

	t.saveToActionContext(businessActionCtx, msg, result)
	return result, nil
}

func (t *TCCRocketMQImpl) Commit(ctx tm.BusinessActionContext) (bool, error) {
	msg, result, err := t.getMessageAndResult(ctx)
	if err != nil {
		return false, err
	}

	if err := t.validateMessage(msg, result); err != nil {
		return false, err
	}

	// TODO: implement rocketmq wrapper to support commit
	log.Info("TCCRocketMQ message commit success xid = ", ctx.Xid, ", branchID = ", ctx.BranchId)
	return true, nil
}

func (t *TCCRocketMQImpl) Rollback(ctx tm.BusinessActionContext) (bool, error) {
	msg, result, err := t.getMessageAndResult(ctx)
	if err != nil {
		log.Error("TCCRocketMQ message rollback: failed to get message or result")
		return true, nil
	}

	if err := t.validateMessage(msg, result); err != nil {
		log.Error("TCCRocketMQ message rollback: invalid message or result")
		return true, nil
	}

	// TODO: implement rocketmq wrapper to support rollback
	log.Info("TCCRocketMQ message rollback success xid = ", ctx.Xid, ", branchID = ", ctx.BranchId)
	return true, nil
}

func (t *TCCRocketMQImpl) saveToActionContext(ctx *tm.BusinessActionContext, msg *primitive.Message, result *primitive.SendResult) {
	ctx.ActionContext[RocketMsgKey] = msg
	ctx.ActionContext[RocketSendResultKey] = result
}

func (t *TCCRocketMQImpl) getMessageAndResult(ctx tm.BusinessActionContext) (*primitive.Message, *primitive.SendResult, error) {
	msg, ok := ctx.ActionContext[RocketMsgKey].(*primitive.Message)
	if !ok {
		return nil, nil, errors.New("invalid message type in context")
	}

	result, ok := ctx.ActionContext[RocketSendResultKey].(*primitive.SendResult)
	if !ok {
		return nil, nil, errors.New("invalid result type in context")
	}

	return msg, result, nil
}

func (t *TCCRocketMQImpl) validateMessage(msg *primitive.Message, result *primitive.SendResult) error {
	if msg == nil {
		return errors.New("message is nil")
	}
	if result == nil {
		return errors.New("send result is nil")
	}
	if strings.TrimSpace(result.OffsetMsgID) == "" || strings.TrimSpace(result.MsgID) == "" {
		return errors.New("invalid message ID")
	}
	return nil
}
