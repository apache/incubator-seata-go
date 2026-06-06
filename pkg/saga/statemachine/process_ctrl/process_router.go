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

package process_ctrl

import (
	"context"

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
)

type RouterHandler interface {
	Route(ctx context.Context, processContext ProcessContext) error
}

type ProcessRouter interface {
	Route(ctx context.Context, processContext ProcessContext) (Instruction, error)
}

type InterceptAbleStateRouter interface {
	StateRouter
	StateRouterInterceptor() []StateRouterInterceptor
	RegistryStateRouterInterceptor(stateRouterInterceptor StateRouterInterceptor)
}

type StateRouter interface {
	Route(ctx context.Context, processContext ProcessContext, state statelang.State) (Instruction, error)
}

type StateRouterInterceptor interface {
	PreRoute(ctx context.Context, processContext ProcessContext, state statelang.State) error
	PostRoute(ctx context.Context, processContext ProcessContext, instruction Instruction, err error) error
	Match(stateType string) bool
}
