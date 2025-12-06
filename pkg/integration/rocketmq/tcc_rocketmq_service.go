/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rocketmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

type TCCRocketMQService struct {
	producer MQProducer
}

func (s *TCCRocketMQService) GetActionName() string {
	return ActionNameTCCRocketMQ
}

func (s *TCCRocketMQService) Prepare(ctx context.Context, msg interface{}) (bool, error) {
	if !tm.IsGlobalTx(ctx) {
		return false, fmt.Errorf("not in global transaction")
	}

	// Send message first to get sendResult
	sendResult, err := s.sendMessageWithBranchRegister(ctx, msg)
	if err != nil {
		return false, err
	}

	// Get businessCtx after registration (it's set by sendMessageWithBranchRegister)
	businessCtx := tm.GetBusinessActionContext(ctx)
	if businessCtx == nil {
		return false, fmt.Errorf("business action context not found after registration")
	}

	log.Infof("RocketMQ message send prepare, xid=%s, branchId=%d, msgId=%s",
		businessCtx.Xid, businessCtx.BranchId, sendResult.(*primitive.SendResult).MsgID)
	return true, nil
}

// sendMessageWithBranchRegister handles branch registration with sendResult in ActionContext
func (s *TCCRocketMQService) sendMessageWithBranchRegister(ctx context.Context, msg interface{}) (interface{}, error) {
	businessCtx := tm.GetBusinessActionContext(ctx)

	// If not registered yet, register first with message info
	if businessCtx == nil {
		// Register branch with message metadata that we have before sending
		if err := s.registerBranchWithMessage(ctx, msg); err != nil {
			return nil, err
		}
		businessCtx = tm.GetBusinessActionContext(ctx)
	}

	// Send the half message
	sendResult, err := s.producer.SendMessageInTransaction(ctx, msg, businessCtx.Xid, businessCtx.BranchId)
	if err != nil {
		return nil, fmt.Errorf("send half message failed: %w", err)
	}

	// Serialize sendResult to make it persistable in ActionContext
	result, ok := sendResult.(*primitive.SendResult)
	if !ok {
		return nil, fmt.Errorf("sendResult type assertion failed")
	}

	// Store serialized sendResult info in ActionContext
	// This will be available in commit/rollback phase after deserialization from TC
	businessCtx.ActionContext[RocketSendResultKey] = map[string]interface{}{
		"MsgID":         result.MsgID,
		"QueueOffset":   result.QueueOffset,
		"TransactionID": result.TransactionID,
		"OffsetMsgID":   result.OffsetMsgID,
		"RegionID":      result.RegionID,
		"Status":        int(result.Status),
	}
	if result.MessageQueue != nil {
		businessCtx.ActionContext["MessageQueue"] = map[string]interface{}{
			"Topic":      result.MessageQueue.Topic,
			"BrokerName": result.MessageQueue.BrokerName,
			"QueueId":    result.MessageQueue.QueueId,
		}
	}

	return sendResult, nil
}

// registerBranchWithMessage registers branch with message metadata
func (s *TCCRocketMQService) registerBranchWithMessage(ctx context.Context, msg interface{}) error {
	actionContext := map[string]interface{}{
		constant.ActionName:      ActionNameTCCRocketMQ,
		constant.PrepareMethod:   "Prepare",
		constant.CommitMethod:    "Commit",
		constant.RollbackMethod:  "Rollback",
		constant.ActionStartTime: time.Now().UnixMilli(),
	}

	// Store message metadata (not the actual message object)
	if mqMsg, ok := msg.(*primitive.Message); ok {
		actionContext[RocketMsgKey] = map[string]interface{}{
			"Topic": mqMsg.Topic,
			"Body":  string(mqMsg.Body),
			"Tags":  mqMsg.GetTags(),
			"Keys":  mqMsg.GetKeys(),
		}
	}

	applicationData, _ := json.Marshal(map[string]interface{}{
		constant.ActionContext: actionContext,
	})

	branchId, err := rm.GetRMRemotingInstance().BranchRegister(rm.BranchRegisterParam{
		BranchType:      branch.BranchTypeTCC,
		ResourceId:      ActionNameTCCRocketMQ,
		Xid:             tm.GetXID(ctx),
		ApplicationData: string(applicationData),
	})
	if err != nil {
		return fmt.Errorf("register tcc branch failed: %w", err)
	}

	businessCtx := &tm.BusinessActionContext{
		Xid:           tm.GetXID(ctx),
		BranchId:      branchId,
		ActionName:    ActionNameTCCRocketMQ,
		ActionContext: actionContext,
	}
	tm.SetBusinessActionContext(ctx, businessCtx)

	return nil
}

func (s *TCCRocketMQService) Commit(ctx context.Context, businessCtx *tm.BusinessActionContext) (bool, error) {
	sendResultData := businessCtx.ActionContext[RocketSendResultKey]

	if sendResultData == nil {
		log.Warnf("sendResult not found in context, rely on RocketMQ CheckLocalTransaction, xid=%s, branchId=%d",
			businessCtx.Xid, businessCtx.BranchId)
		// Return true to mark the branch as committed in Seata
		// RocketMQ will handle the message commit through CheckLocalTransaction callback
		return true, nil
	}

	// Reconstruct SendResult from serialized data
	sendResult, err := s.reconstructSendResult(sendResultData)
	if err != nil {
		log.Errorf("failed to reconstruct sendResult: %v, rely on RocketMQ CheckLocalTransaction, xid=%s, branchId=%d",
			err, businessCtx.Xid, businessCtx.BranchId)
		return true, nil
	}

	// Try to commit the transaction directly
	if err := s.producer.EndTransaction(ctx, sendResult, TransactionCommit); err != nil {
		log.Errorf("commit transaction failed: %v, rely on RocketMQ CheckLocalTransaction, xid=%s, branchId=%d",
			err, businessCtx.Xid, businessCtx.BranchId)
		// Still return true because RocketMQ CheckLocalTransaction will handle it
		return true, nil
	}

	log.Infof("RocketMQ message send commit, xid=%s, branchId=%d", businessCtx.Xid, businessCtx.BranchId)
	return true, nil
}

func (s *TCCRocketMQService) Rollback(ctx context.Context, businessCtx *tm.BusinessActionContext) (bool, error) {
	sendResultData := businessCtx.ActionContext[RocketSendResultKey]

	if sendResultData == nil {
		log.Warnf("sendResult not found in context, rely on RocketMQ CheckLocalTransaction, xid=%s, branchId=%d",
			businessCtx.Xid, businessCtx.BranchId)
		// Return true to mark the branch as rolled back in Seata
		// RocketMQ will handle the message rollback through CheckLocalTransaction callback
		return true, nil
	}

	// Reconstruct SendResult from serialized data
	sendResult, err := s.reconstructSendResult(sendResultData)
	if err != nil {
		log.Errorf("failed to reconstruct sendResult: %v, rely on RocketMQ CheckLocalTransaction, xid=%s, branchId=%d",
			err, businessCtx.Xid, businessCtx.BranchId)
		return true, nil
	}

	// Try to rollback the transaction directly
	if err := s.producer.EndTransaction(ctx, sendResult, TransactionRollback); err != nil {
		log.Errorf("rollback transaction failed: %v, rely on RocketMQ CheckLocalTransaction, xid=%s, branchId=%d",
			err, businessCtx.Xid, businessCtx.BranchId)
		// Still return true because RocketMQ CheckLocalTransaction will handle it
		return true, nil
	}

	log.Infof("RocketMQ message send rollback, xid=%s, branchId=%d", businessCtx.Xid, businessCtx.BranchId)
	return true, nil
}

// reconstructSendResult converts serialized send result data back to primitive.SendResult
func (s *TCCRocketMQService) reconstructSendResult(data interface{}) (*primitive.SendResult, error) {
	resultMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("sendResult data is not a map")
	}

	sendResult := &primitive.SendResult{}

	if msgID, ok := resultMap["MsgID"].(string); ok {
		sendResult.MsgID = msgID
	}
	if offsetMsgID, ok := resultMap["OffsetMsgID"].(string); ok {
		sendResult.OffsetMsgID = offsetMsgID
	}
	if transactionID, ok := resultMap["TransactionID"].(string); ok {
		sendResult.TransactionID = transactionID
	}
	if regionID, ok := resultMap["RegionID"].(string); ok {
		sendResult.RegionID = regionID
	}
	if queueOffset, ok := resultMap["QueueOffset"].(float64); ok {
		sendResult.QueueOffset = int64(queueOffset)
	} else if queueOffset, ok := resultMap["QueueOffset"].(int64); ok {
		sendResult.QueueOffset = queueOffset
	}
	if status, ok := resultMap["Status"].(float64); ok {
		sendResult.Status = primitive.SendStatus(int(status))
	} else if status, ok := resultMap["Status"].(int); ok {
		sendResult.Status = primitive.SendStatus(status)
	}

	return sendResult, nil
}
