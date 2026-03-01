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

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
)

type StatusDecisionStrategy interface {
	// DecideOnEndState Determine state machine execution status when executing to EndState
	DecideOnEndState(ctx context.Context, processContext process_ctrl.ProcessContext,
		stateMachineInstance statelang.StateMachineInstance, exp error) error
	// DecideOnTaskStateFail Determine state machine execution status when executing TaskState error
	DecideOnTaskStateFail(ctx context.Context, processContext process_ctrl.ProcessContext,
		stateMachineInstance statelang.StateMachineInstance, exp error) error
	// DecideMachineForwardExecutionStatus Determine the forward execution state of the state machine
	DecideMachineForwardExecutionStatus(ctx context.Context,
		stateMachineInstance statelang.StateMachineInstance, exp error, specialPolicy bool) error
}
