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

package grpc

import (
	"context"
	"fmt"
	"time"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/backoff"
	"seata.apache.org/seata-go/pkg/util/log"

	"github.com/pkg/errors"
)

type GrpcGlobalTransactionManager struct{}

// Begin a global transaction with given timeout and given name.
func (g *GrpcGlobalTransactionManager) Begin(ctx context.Context, timeout time.Duration) error {
	req := &pb.GlobalBeginRequestProto{
		AbstractTransactionRequest: &pb.AbstractTransactionRequestProto{
			AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_BEGIN},
		},
		Timeout:         int32(timeout.Milliseconds()),
		TransactionName: tm.GetTxName(ctx),
	}
	res, err := grpc.GetGrpcRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("GlobalBeginRequest  error %v", err)
		return err
	}
	if res == nil || res.(*pb.GlobalBeginResponseProto).
		AbstractTransactionResponse.
		AbstractResultMessage.ResultCode == pb.ResultCodeProto(message.ResultCodeFailed) {
		log.Errorf("GlobalBeginRequest result is empty or result code is failed, res %v", res)
		return fmt.Errorf("GlobalBeginRequest result is empty or result code is failed.")
	}
	log.Infof("GlobalBeginRequest success, res %v", res)

	tm.SetXID(ctx, res.(*pb.GlobalBeginResponseProto).Xid)
	return nil
}

// Commit the global transaction.
func (g *GrpcGlobalTransactionManager) Commit(ctx context.Context, gtr *tm.GlobalTransaction) error {
	if tm.IsTimeout(ctx) {
		log.Infof("Rollback: tm detected timeout in global gtr %s", gtr.Xid)
		if err := tm.GetGlobalTransactionManager().Rollback(ctx, gtr); err != nil {
			log.Errorf("Rollback transaction failed, error: %v in global gtr % s", err, gtr.Xid)
			return err
		}
		return nil
	}
	if gtr.TxRole != tm.Launcher {
		log.Infof("Ignore Commit(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return fmt.Errorf("Commit xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: tm.GetTmConfig().CommitRetryCount,
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: 200 * time.Millisecond,
	})

	req := &pb.GlobalCommitRequestProto{
		AbstractGlobalEndRequest: &pb.AbstractGlobalEndRequestProto{
			AbstractTransactionRequest: &pb.AbstractTransactionRequestProto{
				AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_BEGIN},
			},
			Xid: gtr.Xid,
		},
	}
	var res interface{}
	var err error
	for bf.Ongoing() {
		if res, err = grpc.GetGrpcRemotingClient().SendSyncRequest(req); err == nil {
			break
		}
		log.Warnf("send global commit request failed, xid %s, error %v", gtr.Xid, err)
		bf.Wait()
	}

	if err != nil || bf.Err() != nil {
		lastErr := errors.Wrap(err, bf.Err().Error())
		log.Warnf("send global commit request failed, xid %s, error %v", gtr.Xid, lastErr)
		return lastErr
	}

	log.Infof("send global commit request success, xid %s", gtr.Xid)
	gtr.TxStatus = message.GlobalStatus(res.(*pb.GlobalCommitResponseProto).AbstractGlobalEndResponse.GlobalStatus)

	return nil
}

// Rollback the global transaction.
func (g *GrpcGlobalTransactionManager) Rollback(ctx context.Context, gtr *tm.GlobalTransaction) error {
	if gtr.TxRole != tm.Launcher {
		log.Infof("Ignore Rollback(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return fmt.Errorf("Rollback xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: tm.GetTmConfig().RollbackRetryCount,
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: 200 * time.Millisecond,
	})

	var res interface{}
	req := &pb.GlobalRollbackRequestProto{
		AbstractGlobalEndRequest: &pb.AbstractGlobalEndRequestProto{
			AbstractTransactionRequest: &pb.AbstractTransactionRequestProto{
				AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_ROLLBACK},
			},
			Xid: gtr.Xid,
		},
	}

	var err error
	for bf.Ongoing() {
		if res, err = grpc.GetGrpcRemotingClient().SendSyncRequest(req); err == nil {
			break
		}
		log.Errorf("GlobalRollbackRequest rollback failed, xid %s, error %v", gtr.Xid, err)
		bf.Wait()
	}

	if err != nil && bf.Err() != nil {
		lastErr := errors.Wrap(err, bf.Err().Error())
		log.Errorf("GlobalRollbackRequest rollback failed, xid %s, error %v", gtr.Xid, lastErr)
		return lastErr
	}

	log.Infof("GlobalRollbackRequest rollback success, xid %s,", gtr.Xid)
	gtr.TxStatus = message.GlobalStatus(res.(*pb.GlobalRollbackResponseProto).AbstractGlobalEndResponse.GlobalStatus)

	return nil
}
