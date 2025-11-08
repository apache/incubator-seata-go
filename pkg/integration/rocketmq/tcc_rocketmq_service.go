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

	businessCtx := tm.GetBusinessActionContext(ctx)
	if businessCtx == nil {
		if err := s.registerBranch(ctx, msg); err != nil {
			return false, err
		}
		businessCtx = tm.GetBusinessActionContext(ctx)
	}

	sendResult, err := s.producer.SendMessageInTransaction(ctx, msg, businessCtx.Xid, businessCtx.BranchId)
	if err != nil {
		return false, fmt.Errorf("send half message failed: %w", err)
	}

	businessCtx.ActionContext[RocketMsgKey] = msg
	businessCtx.ActionContext[RocketSendResultKey] = sendResult

	log.Infof("RocketMQ message send prepare, xid=%s, branchId=%d", businessCtx.Xid, businessCtx.BranchId)
	return true, nil
}

func (s *TCCRocketMQService) Commit(ctx context.Context, businessCtx *tm.BusinessActionContext) (bool, error) {
	msg := businessCtx.ActionContext[RocketMsgKey]
	sendResult := businessCtx.ActionContext[RocketSendResultKey]

	if msg == nil || sendResult == nil {
		log.Warnf("message or sendResult not found in context, xid=%s, branchId=%d", businessCtx.Xid, businessCtx.BranchId)
		return true, nil
	}

	if err := s.producer.EndTransaction(ctx, sendResult, TransactionCommit); err != nil {
		return false, fmt.Errorf("commit transaction failed: %w", err)
	}

	log.Infof("RocketMQ message send commit, xid=%s, branchId=%d", businessCtx.Xid, businessCtx.BranchId)
	return true, nil
}

func (s *TCCRocketMQService) Rollback(ctx context.Context, businessCtx *tm.BusinessActionContext) (bool, error) {
	msg := businessCtx.ActionContext[RocketMsgKey]
	sendResult := businessCtx.ActionContext[RocketSendResultKey]

	if msg == nil || sendResult == nil {
		log.Warnf("message or sendResult not found, xid=%s, branchId=%d", businessCtx.Xid, businessCtx.BranchId)
		return true, nil
	}

	if err := s.producer.EndTransaction(ctx, sendResult, TransactionRollback); err != nil {
		return false, fmt.Errorf("rollback transaction failed: %w", err)
	}

	log.Infof("RocketMQ message send rollback, xid=%s, branchId=%d", businessCtx.Xid, businessCtx.BranchId)
	return true, nil
}

func (s *TCCRocketMQService) registerBranch(ctx context.Context, msg interface{}) error {
	actionContext := map[string]interface{}{
		constant.ActionName:      ActionNameTCCRocketMQ,
		constant.PrepareMethod:   "Prepare",
		constant.CommitMethod:    "Commit",
		constant.RollbackMethod:  "Rollback",
		constant.ActionStartTime: time.Now().UnixMilli(),
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
