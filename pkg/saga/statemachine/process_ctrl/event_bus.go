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

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/v2/pkg/util/collection"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type EventBus interface {
	Run(ctx context.Context, processContext ProcessContext) error
}

type DirectEventBus struct {
	handler       ProcessHandler
	processRouter ProcessRouter
}

func NewDirectEventBus(handler ProcessHandler, router ProcessRouter) *DirectEventBus {
	return &DirectEventBus{handler: handler, processRouter: router}
}

func (d *DirectEventBus) Run(ctx context.Context, processContext ProcessContext) error {
	// Use stack to avoid deep recursion (same as original DirectEventBus.Offer)
	stack := collection.NewStack()
	processContext.SetVariable(constant.VarNameSyncExeStack, stack)
	stack.Push(processContext)

	for stack.Len() > 0 {
		current := stack.Pop().(ProcessContext)
		if err := d.handler.Process(ctx, current); err != nil {
			return err
		}
		instruction, err := d.processRouter.Route(ctx, current)
		if err != nil {
			return err
		}
		if instruction != nil {
			current.SetInstruction(instruction)
			stack.Push(current)
		}
	}
	return nil
}

type AsyncEventBus struct {
	handler       ProcessHandler
	processRouter ProcessRouter
}

func NewAsyncEventBus(handler ProcessHandler, router ProcessRouter) *AsyncEventBus {
	return &AsyncEventBus{handler: handler, processRouter: router}
}

func (a *AsyncEventBus) Run(ctx context.Context, processContext ProcessContext) error {
	go func() {
		stack := collection.NewStack()
		stack.Push(processContext)
		for stack.Len() > 0 {
			current := stack.Pop().(ProcessContext)
			if err := a.handler.Process(ctx, current); err != nil {
				log.Errorf("async process error: %s", err.Error())
				return
			}
			instruction, err := a.processRouter.Route(ctx, current)
			if err != nil {
				log.Errorf("async route error: %s", err.Error())
				return
			}
			if instruction != nil {
				current.SetInstruction(instruction)
				stack.Push(current)
			}
		}
	}()
	return nil
}
