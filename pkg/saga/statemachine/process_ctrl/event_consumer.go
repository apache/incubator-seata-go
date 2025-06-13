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
	"fmt"
	"github.com/pkg/errors"
)

type EventConsumer interface {
	Accept(event Event) bool

	Process(ctx context.Context, event Event) error
}

type ProcessCtrlEventConsumer struct {
	processController ProcessController
}

func (p ProcessCtrlEventConsumer) Accept(event Event) bool {
	if event == nil {
		return false
	}

	_, ok := event.(ProcessContext)
	return ok
}

func (p ProcessCtrlEventConsumer) Process(ctx context.Context, event Event) error {
	processContext, ok := event.(ProcessContext)
	if !ok {
		return errors.New(fmt.Sprint("event %T is illegal, required process_ctrl.ProcessContext", event))
	}
	return p.processController.Process(ctx, processContext)
}
