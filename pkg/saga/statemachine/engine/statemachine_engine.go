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

package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineEngine interface {
	// Start starts a state machine instance
	Start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error)
	// StartAsync start a state machine instance asynchronously
	StartAsync(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{},
		callback CallBack) (statelang.StateMachineInstance, error)
	// StartWithBusinessKey starts a state machine instance with a business key
	StartWithBusinessKey(ctx context.Context, stateMachineName string, tenantId string, businessKey string,
		startParams map[string]interface{}) (statelang.StateMachineInstance, error)
	// StartWithBusinessKeyAsync starts a state machine instance with a business key asynchronously
	StartWithBusinessKeyAsync(ctx context.Context, stateMachineName string, tenantId string, businessKey string,
		startParams map[string]interface{}, callback CallBack) (statelang.StateMachineInstance, error)
	// Forward  restart a failed state machine instance
	Forward(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}) (statelang.StateMachineInstance, error)
	// ForwardAsync restart a failed state machine instance asynchronously
	ForwardAsync(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}, callback CallBack) (statelang.StateMachineInstance, error)
	// Compensate compensate a state machine instance
	Compensate(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}) (statelang.StateMachineInstance, error)
	// CompensateAsync compensate a state machine instance asynchronously
	CompensateAsync(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}, callback CallBack) (statelang.StateMachineInstance, error)
	// SkipAndForward skips the current failed state instance and restarts the state machine instance
	SkipAndForward(ctx context.Context, stateMachineInstId string, replaceParams map[string]interface{}) (statelang.StateMachineInstance, error)
	// SkipAndForwardAsync skips the current failed state instance and restarts the state machine instance asynchronously
	SkipAndForwardAsync(ctx context.Context, stateMachineInstId string, callback CallBack) (statelang.StateMachineInstance, error)
	// GetStateMachineConfig gets the state machine configurations
	GetStateMachineConfig() StateMachineConfig
	// ReloadStateMachineInstance reloads a state machine instance
	ReloadStateMachineInstance(ctx context.Context, instId string) (statelang.StateMachineInstance, error)
}

type CallBack interface {
	OnFinished(ctx context.Context, context process_ctrl.ProcessContext, stateMachineInstance statelang.StateMachineInstance)
	OnError(ctx context.Context, context process_ctrl.ProcessContext, stateMachineInstance statelang.StateMachineInstance, err error)
}
