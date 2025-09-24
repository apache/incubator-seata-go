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
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/seata/seata-go/pkg/saga/statemachine/engine/config"
	engExc "github.com/seata/seata-go/pkg/saga/statemachine/engine/exception"
	"github.com/seata/seata-go/pkg/util/errors"
)

func TestStartAsyncDisabled(t *testing.T) {
	cfg, err := config.NewDefaultStateMachineConfig(
		config.WithEnableAsync(false),
		config.WithStateMachineResources(nil),
	)
	require.NoError(t, err)

	engine := &ProcessCtrlStateMachineEngine{StateMachineConfig: cfg}

	_, err = engine.StartAsync(context.Background(), "dummy", "", nil, nil)
	require.Error(t, err)

	execErr, ok := engExc.IsEngineExecutionException(err)
	if !ok {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
	require.Equal(t, errors.AsynchronousStartDisabled, execErr.Code)
}

func TestStartAsyncEnabledWithoutStateMachine(t *testing.T) {
	cfg, err := config.NewDefaultStateMachineConfig(
		config.WithEnableAsync(true),
		config.WithStateMachineResources(nil),
	)
	require.NoError(t, err)

	engine := &ProcessCtrlStateMachineEngine{StateMachineConfig: cfg}

	_, err = engine.StartAsync(context.Background(), "missing", "", nil, nil)
	require.Error(t, err)

	execErr, ok := engExc.IsEngineExecutionException(err)
	if !ok {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
	require.Equal(t, errors.ObjectNotExists, execErr.Code)
}
