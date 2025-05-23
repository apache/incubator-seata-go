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

package core

import (
	"github.com/seata/seata-go/pkg/saga/statemachine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"sync"
)

type StateMachineConfig interface {
	StateLogRepository() StateLogRepository

	StateMachineRepository() StateMachineRepository

	StateLogStore() StateLogStore

	StateLangStore() StateLangStore

	ExpressionFactoryManager() *expr.ExpressionFactoryManager

	ExpressionResolver() expr.ExpressionResolver

	SeqGenerator() sequence.SeqGenerator

	StatusDecisionStrategy() StatusDecisionStrategy

	EventPublisher() EventPublisher

	AsyncEventPublisher() EventPublisher

	ServiceInvokerManager() invoker.ServiceInvokerManager

	ScriptInvokerManager() invoker.ScriptInvokerManager

	CharSet() string

	GetDefaultTenantId() string

	GetTransOperationTimeout() int

	GetServiceInvokeTimeout() int

	ComponentLock() *sync.Mutex

	RegisterStateMachineDef(resources []string) error

	RegisterExpressionFactory(expressionType string, factory expr.ExpressionFactory)

	RegisterServiceInvoker(serviceType string, invoker invoker.ServiceInvoker)

	GetStateMachineDefinition(name string) *statemachine.StateMachineObject

	GetExpressionFactory(expressionType string) expr.ExpressionFactory

	GetServiceInvoker(serviceType string) invoker.ServiceInvoker
}
