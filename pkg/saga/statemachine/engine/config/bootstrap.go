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

package config

import (
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/pcext"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl/handlers"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl/process"
)

// bootstrapProcessWiring wires default handlers/routers and router handler into BusinessProcessor
func (c *DefaultStateMachineConfig) bootstrapProcessWiring() error {
	// Process handler for state machine
	smHandler := pcext.NewStateMachineProcessHandler()
	// Register ServiceTask handler by default
	smHandler.RegistryStateHandler(constant.StateTypeServiceTask, handlers.NewServiceTaskStateHandler())

	// Router for state machine
	smRouter := &pcext.StateMachineProcessRouter{}
	smRouter.InitDefaultStateRouters()

	// Default router handler binds event publisher and process routers
	drh := &process_ctrl.DefaultRouterHandler{}
	drh.SetEventPublisher(c.EventPublisher())
	drh.SetProcessRouters(map[string]process_ctrl.ProcessRouter{
		string(process.StateLang): smRouter,
	})

	// Register into BusinessProcessor
	pcImpl, ok := c.processController.(*process_ctrl.ProcessControllerImpl)
	if !ok {
		return fmt.Errorf("ProcessController is not an instance of ProcessControllerImpl")
	}
	bp := pcImpl.BusinessProcessor()
	// need concrete DefaultBusinessProcessor to call Registry APIs
	dbp, ok := bp.(*process_ctrl.DefaultBusinessProcessor)
	if !ok {
		return fmt.Errorf("BusinessProcessor is not DefaultBusinessProcessor, got %T", bp)
	}
	dbp.RegistryProcessHandler(process.StateLang, smHandler)
	dbp.RegistryRouterHandler(process.StateLang, drh)
	return nil
}
