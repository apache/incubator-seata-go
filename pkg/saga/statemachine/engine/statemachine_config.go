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
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/repo"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/store"
	"sync"
)

type StateMachineConfig interface {
	StateLogRepository() repo.StateLogRepository

	StateMachineRepository() repo.StateMachineRepository

	StateLogStore() store.StateLogStore

	StateLangStore() store.StateLangStore

	ExpressionFactoryManager() expr.ExpressionFactoryManager

	ExpressionResolver() expr.ExpressionResolver

	SeqGenerator() sequence.SeqGenerator

	StatusDecisionStrategy() StatusDecisionStrategy

	EventPublisher() process_ctrl.EventPublisher

	AsyncEventPublisher() process_ctrl.EventPublisher

	ServiceInvokerManager() invoker.ServiceInvokerManager

	ScriptInvokerManager() invoker.ScriptInvokerManager

	CharSet() string

	DefaultTenantId() string

	TransOperationTimeout() int

	ServiceInvokeTimeout() int

	ComponentLock() *sync.Mutex
}
