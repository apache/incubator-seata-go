package rocketmq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/rm/tcc"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

type TCCRocketMQImpl struct {
	producer    MessageSender
	resourceMgr rm.ResourceManager
	tccResource *tcc.TCCResource
}

func NewTCCRocketMQImpl(producer MessageSender) (*TCCRocketMQImpl, error) {
	if producer == nil {
		return nil, errors.New("producer cannot be nil")
	}

	impl := &TCCRocketMQImpl{
		producer: producer,
	}

	tccResource, err := impl.createTCCResource()
	if err != nil {
		return nil, fmt.Errorf("failed to create TCC resource: %w", err)
	}

	impl.tccResource = tccResource

	if err := impl.registerTCCResource(); err != nil {
		return nil, fmt.Errorf("failed to register TCC resource: %w", err)
	}

	return impl, nil
}

func (t *TCCRocketMQImpl) createTCCResource() (*tcc.TCCResource, error) {
	return tcc.ParseTCCResource(t)
}

func (t *TCCRocketMQImpl) registerTCCResource() error {
	resourceMgr := rm.GetRmCacheInstance().GetResourceManager(t.GetBranchType())
	if resourceMgr == nil {
		return errors.New("resource manager not initialized")
	}

	return resourceMgr.RegisterResource(t.tccResource)
}

func (t *TCCRocketMQImpl) Prepare(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	log.Infof("TCCRocketMQ Prepare phase started")

	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	if !tm.IsGlobalTx(ctx) {
		return nil, errors.New("not in global transaction context")
	}

	globalTx := tm.GetTx(ctx)
	if globalTx == nil {
		return nil, errors.New("global transaction not found")
	}

	result, err := t.producer.SendTransactionMessage(ctx, msg)
	if err != nil {
		log.Errorf("Failed to send transaction message: %v", err)
		return nil, err
	}

	branchID, err := t.registerBranchTransaction(ctx, globalTx.Xid, msg, result)
	if err != nil {
		log.Errorf("Failed to register branch transaction: %v", err)
		return nil, err
	}

	actionCtx := tm.BusinessActionContext{
		Xid:      globalTx.Xid,
		BranchId: branchID,
		ActionContext: map[string]interface{}{
			RocketMsgKey:        msg,
			RocketSendResultKey: result,
			RocketTopicKey:      msg.Topic,
			RocketTagsKey:       msg.GetTags(),
		},
	}

	tm.SetBusinessActionContext(ctx, &actionCtx)

	log.Infof("TCCRocketMQ Prepare phase completed successfully, xid=%s, branchId=%d",
		globalTx.Xid, branchID)

	return result, nil
}

func (t *TCCRocketMQImpl) registerBranchTransaction(ctx context.Context, xid string,
	msg *primitive.Message, result *primitive.SendResult) (int64, error) {

	resourceId := t.buildResourceId(msg.Topic, msg.GetTags())

	resourceMgr := rm.GetRmCacheInstance().GetResourceManager(t.GetBranchType())
	if resourceMgr == nil {
		return 0, errors.New("resource manager not initialized")
	}

	branchRegisterParam := rm.BranchRegisterParam{
		Xid:             xid,
		ResourceId:      resourceId,
		BranchType:      RocketBranchType,
		ApplicationData: string(t.buildApplicationData(msg, result)),
		ClientId:        "", // TODO: add client id
		LockKeys:        "", // TODO: add lock key
	}

	branchRegisterResponse, err := resourceMgr.BranchRegister(ctx, branchRegisterParam)
	if err != nil {
		return 0, fmt.Errorf("failed to register branch transaction: %w", err)
	}

	return branchRegisterResponse, nil
}

func (t *TCCRocketMQImpl) buildResourceId(topic, tags string) string {
	if tags != "" {
		return fmt.Sprintf("%s:%s:%s", RocketTccResourceName, topic, tags)
	}
	return fmt.Sprintf("%s:%s", RocketTccResourceName, topic)
}

func (t *TCCRocketMQImpl) buildApplicationData(msg *primitive.Message, result *primitive.SendResult) []byte {
	data := map[string]string{
		"topic":    msg.Topic,
		"tags":     msg.GetTags(),
		"keys":     msg.GetKeys(),
		"msgId":    result.MsgID,
		"offsetId": result.OffsetMsgID,
		"queueId":  strconv.Itoa(result.MessageQueue.QueueId),
	}

	var parts []string
	for k, v := range data {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	return []byte(strings.Join(parts, "&"))
}

func (t *TCCRocketMQImpl) Commit(ctx tm.BusinessActionContext) (bool, error) {
	log.Infof("TCCRocketMQ Commit phase started, xid=%s, branchId=%d",
		ctx.Xid, ctx.BranchId)

	msg, result, err := t.getMessageAndResult(ctx)
	if err != nil {
		log.Errorf("Failed to get message or result from context: %v", err)
		return true, nil
	}

	if err := t.validateMessageAndResult(msg, result); err != nil {
		log.Errorf("Invalid message or result: %v", err)
		return true, nil
	}

	log.Infof("TCCRocketMQ message commit completed, xid=%s, branchId=%d, msgId=%s",
		ctx.Xid, ctx.BranchId, result.MsgID)

	return true, nil
}

func (t *TCCRocketMQImpl) Rollback(ctx tm.BusinessActionContext) (bool, error) {
	log.Infof("TCCRocketMQ Rollback phase started, xid=%s, branchId=%d",
		ctx.Xid, ctx.BranchId)

	msg, result, err := t.getMessageAndResult(ctx)
	if err != nil {
		log.Warnf("Failed to get message or result from context during rollback: %v", err)
		return true, nil
	}

	if err := t.validateMessageAndResult(msg, result); err != nil {
		log.Warnf("Invalid message or result during rollback: %v", err)
		return true, nil
	}

	log.Infof("TCCRocketMQ message rollback completed, xid=%s, branchId=%d, msgId=%s",
		ctx.Xid, ctx.BranchId, result.MsgID)

	return true, nil
}

func (t *TCCRocketMQImpl) GetActionName() string {
	return RocketTccActionName
}

func (t *TCCRocketMQImpl) getMessageAndResult(ctx tm.BusinessActionContext) (
	*primitive.Message, *primitive.SendResult, error) {

	msgInterface, exists := ctx.ActionContext[RocketMsgKey]
	if !exists {
		return nil, nil, errors.New("message not found in context")
	}

	msg, ok := msgInterface.(*primitive.Message)
	if !ok {
		return nil, nil, errors.New(ErrMsgInvalidMessageType)
	}

	resultInterface, exists := ctx.ActionContext[RocketSendResultKey]
	if !exists {
		return nil, nil, errors.New("send result not found in context")
	}

	result, ok := resultInterface.(*primitive.SendResult)
	if !ok {
		return nil, nil, errors.New(ErrMsgInvalidResultType)
	}

	return msg, result, nil
}

func (t *TCCRocketMQImpl) validateMessageAndResult(msg *primitive.Message, result *primitive.SendResult) error {
	if msg == nil {
		return errors.New(ErrMsgInvalidMessage)
	}

	if result == nil {
		return errors.New(ErrMsgInvalidResult)
	}

	if strings.TrimSpace(result.MsgID) == "" {
		return errors.New(ErrMsgInvalidMessageID)
	}

	return nil
}

func (t *TCCRocketMQImpl) GetResourceId() string {
	return RocketTccResourceName
}

func (t *TCCRocketMQImpl) GetBranchType() branch.BranchType {
	return RocketBranchType
}
