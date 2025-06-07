package rocketmq

import (
	"context"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/pkg/errors"
	"strings"
	"sync"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

type MQProducer struct {
	transactionProducer rocketmq.TransactionProducer
	normalProducer      rocketmq.Producer
	transactionListener *TransactionListener
	config              *Config
	tccImpl             *TCCRocketMQImpl
	mutex               sync.RWMutex
	started             bool
}

func NewMQProducer(ctx context.Context, config *Config) (*MQProducer, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	pd := &MQProducer{
		config: config,
	}

	tm.InitSeataContext(ctx)
	pd.transactionListener = NewTransactionListener(ctx, pd)

	transactionProducer, err := pd.createTransactionProducer()
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction pd: %w", err)
	}
	pd.transactionProducer = transactionProducer

	normalProducer, err := pd.createNormalProducer()
	if err != nil {
		return nil, fmt.Errorf("failed to create normal pd: %w", err)
	}
	pd.normalProducer = normalProducer

	tccImpl, err := NewTCCRocketMQImpl(pd)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCC implementation: %w", err)
	}
	pd.tccImpl = tccImpl

	return pd, nil
}

func (p *MQProducer) createTransactionProducer() (rocketmq.TransactionProducer, error) {
	opts := []producer.Option{
		producer.WithNameServer([]string{p.config.Addr}),
		producer.WithGroupName(p.config.GroupName),
		producer.WithRetry(p.config.RetryTimes),
		producer.WithSendMsgTimeout(p.config.SendTimeout),
	}

	if p.config.Namespace != "" {
		opts = append(opts, producer.WithNamespace(p.config.Namespace))
	}

	if p.config.InstanceName != "" {
		opts = append(opts, producer.WithInstanceName(p.config.InstanceName))
	}

	return rocketmq.NewTransactionProducer(p.transactionListener, opts...)
}

func (p *MQProducer) createNormalProducer() (rocketmq.Producer, error) {
	opts := []producer.Option{
		producer.WithNameServer([]string{p.config.Addr}),
		producer.WithGroupName(p.config.GroupName + "_NORMAL"),
		producer.WithRetry(p.config.RetryTimes),
		producer.WithSendMsgTimeout(p.config.SendTimeout),
	}

	if p.config.Namespace != "" {
		opts = append(opts, producer.WithNamespace(p.config.Namespace))
	}

	if p.config.InstanceName != "" {
		opts = append(opts, producer.WithInstanceName(p.config.InstanceName+"_NORMAL"))
	}

	return rocketmq.NewProducer(opts...)
}

func (p *MQProducer) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.started {
		return nil
	}

	if err := p.transactionProducer.Start(); err != nil {
		return fmt.Errorf("failed to start transaction producer: %w", err)
	}

	if err := p.normalProducer.Start(); err != nil {
		_ = p.transactionProducer.Shutdown()
		return fmt.Errorf("failed to start normal producer: %w", err)
	}

	p.started = true
	log.Infof("MQProducer started successfully")

	return nil
}

func (p *MQProducer) Shutdown() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.started {
		return nil
	}

	var errs []error

	if err := p.normalProducer.Shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown normal producer: %w", err))
	}

	if err := p.transactionProducer.Shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown transaction producer: %w", err))
	}

	p.started = false

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	log.Infof("MQProducer shutdown successfully")
	return nil
}

func (p *MQProducer) SendMessage(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	if !p.started {
		return nil, errors.New("producer not started")
	}

	msg.Topic = p.withNamespace(msg.Topic)

	return p.normalProducer.SendSync(ctx, msg)
}

func (p *MQProducer) SendTransactionMessage(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	if !p.started {
		return nil, errors.New("producer not started")
	}

	if !tm.IsGlobalTx(ctx) {
		return nil, errors.New("not in global transaction context")
	}

	globalTx := tm.GetTx(ctx)
	if globalTx == nil {
		return nil, errors.New("global transaction not found")
	}

	if err := p.prepareTransactionMessage(msg, globalTx.Xid); err != nil {
		return nil, err
	}

	result, err := p.transactionProducer.SendMessageInTransaction(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction message: %w", err)
	}

	return result.SendResult, nil
}

func (p *MQProducer) prepareTransactionMessage(msg *primitive.Message, xid string) error {
	msg.Topic = p.withNamespace(msg.Topic)

	msg.RemoveProperty(primitive.PropertyDelayTimeLevel)

	if strings.TrimSpace(msg.Topic) == "" {
		return errors.New("topic cannot be empty")
	}

	msg.WithProperties(map[string]string{
		PropertySeataXID:    xid,
		PropertySeataAction: TccPrepareMethod,
	})

	return nil
}

func (p *MQProducer) withNamespace(topic string) string {
	if p.config.Namespace == "" || strings.Contains(topic, "%") {
		return topic
	}
	return p.config.Namespace + "%" + topic
}

func (p *MQProducer) GetTCCImpl() *TCCRocketMQImpl {
	return p.tccImpl
}

type TransactionListener struct {
	ctx      context.Context
	producer *MQProducer
}

func NewTransactionListener(ctx context.Context, producer *MQProducer) *TransactionListener {
	return &TransactionListener{
		ctx:      ctx,
		producer: producer,
	}
}

func (l *TransactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	xid := msg.GetProperty(PropertySeataXID)
	log.Infof("ExecuteLocalTransaction: xid=%s, msgId=%s", xid, msg.GetProperty("__MESSAGE_ID__"))

	return primitive.UnknowState
}

func (l *TransactionListener) CheckLocalTransaction(msgExt *primitive.MessageExt) primitive.LocalTransactionState {
	xid := msgExt.GetProperty(PropertySeataXID)

	log.Infof("CheckLocalTransaction: xid=%s, msgId=%s", xid, msgExt.MsgId)

	if !l.isValidXID(xid) {
		log.Warnf("Invalid XID: %s", xid)
		tm.SetTxStatus(l.ctx, message.GlobalStatusFinished)
		return primitive.RollbackMessageState
	}

	globalStatus := tm.GetTxStatus(l.ctx)

	return l.determineTransactionState(*globalStatus)
}

func (l *TransactionListener) isValidXID(xid string) bool {
	return len(strings.TrimSpace(xid)) > 0
}

func (l *TransactionListener) determineTransactionState(status message.GlobalStatus) primitive.LocalTransactionState {
	tm.SetTxStatus(l.ctx, status)
	if _, exists := GlobalStatusMaps.Commit[status]; exists {
		return primitive.CommitMessageState
	}

	if _, exists := GlobalStatusMaps.Rollback[status]; exists {
		return primitive.RollbackMessageState
	}

	if l.isTimeoutStatus(status) {
		return primitive.RollbackMessageState
	}

	if status == message.GlobalStatusFinished {
		return primitive.CommitMessageState
	}

	return primitive.UnknowState
}

func (l *TransactionListener) isTimeoutStatus(status message.GlobalStatus) bool {
	timeoutStatuses := []message.GlobalStatus{
		message.GlobalStatusTimeoutRollbacking,
		message.GlobalStatusTimeoutRollbackRetrying,
		message.GlobalStatusTimeoutRollbacked,
		message.GlobalStatusTimeoutRollbackFailed,
	}

	for _, s := range timeoutStatuses {
		if status == s {
			return true
		}
	}
	return false
}
