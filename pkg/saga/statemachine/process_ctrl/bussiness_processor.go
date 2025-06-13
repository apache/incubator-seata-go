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
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl/process"
	"sync"
)

type BusinessProcessor interface {
	Process(ctx context.Context, processContext ProcessContext) error

	Route(ctx context.Context, processContext ProcessContext) error
}

type DefaultBusinessProcessor struct {
	processHandlers map[string]ProcessHandler
	routerHandlers  map[string]RouterHandler
	mu              sync.RWMutex
}

func (d *DefaultBusinessProcessor) RegistryProcessHandler(processType process.ProcessType, processHandler ProcessHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.processHandlers[string(processType)] = processHandler
}

func (d *DefaultBusinessProcessor) RegistryRouterHandler(processType process.ProcessType, routerHandler RouterHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.routerHandlers[string(processType)] = routerHandler
}

func (d *DefaultBusinessProcessor) Process(ctx context.Context, processContext ProcessContext) error {
	processType := d.matchProcessType(processContext)

	processHandler, err := d.getProcessHandler(processType)
	if err != nil {
		return err
	}

	return processHandler.Process(ctx, processContext)
}

func (d *DefaultBusinessProcessor) Route(ctx context.Context, processContext ProcessContext) error {
	processType := d.matchProcessType(processContext)

	routerHandler, err := d.getRouterHandler(processType)
	if err != nil {
		return err
	}

	return routerHandler.Route(ctx, processContext)
}

func (d *DefaultBusinessProcessor) getProcessHandler(processType process.ProcessType) (ProcessHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	processHandler, ok := d.processHandlers[string(processType)]
	if !ok {
		return nil, errors.New("Cannot find Process handler by type " + string(processType))
	}
	return processHandler, nil
}

func (d *DefaultBusinessProcessor) getRouterHandler(processType process.ProcessType) (RouterHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	routerHandler, ok := d.routerHandlers[string(processType)]
	if !ok {
		return nil, errors.New("Cannot find router handler by type " + string(processType))
	}
	return routerHandler, nil
}

func (d *DefaultBusinessProcessor) matchProcessType(processContext ProcessContext) process.ProcessType {
	ok := processContext.HasVariable(constant.VarNameProcessType)
	if ok {
		return processContext.GetVariable(constant.VarNameProcessType).(process.ProcessType)
	}
	return process.StateLang
}
