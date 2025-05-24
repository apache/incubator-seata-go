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
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/pkg/tm"
)

// TCCRocketMQ defines the interface for TCC transaction operations
type TCCRocketMQ interface {
	// SetProducer sets the MQ producer
	SetProducer(producer *MQProducer)

	// Prepare handles the prepare phase of TCC transaction
	Prepare(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error)

	// Commit handles the commit phase of TCC transaction
	Commit(ctx tm.BusinessActionContext) (bool, error)

	// Rollback handles the rollback phase of TCC transaction
	Rollback(ctx tm.BusinessActionContext) (bool, error)
}
