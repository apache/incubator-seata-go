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

package engine_test

import (
	"context"
	"testing"

	enginepkg "seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/core"
)

func TestProcessCtrlEngineInitializes(t *testing.T) {
	eng, err := core.NewProcessCtrlStateMachineEngine()
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}
	if eng.GetStateMachineConfig() == nil {
		t.Fatalf("state machine config should not be nil")
	}
	if _, ok := interface{}(eng).(enginepkg.StateMachineEngine); !ok {
		t.Fatalf("ProcessCtrlStateMachineEngine should satisfy engine.StateMachineEngine")
	}
}

func TestProcessCtrlEngineStartMissingDefinitionFails(t *testing.T) {
	eng, err := core.NewProcessCtrlStateMachineEngine()
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}
	if _, err = eng.Start(context.Background(), "undefined", "", nil); err == nil {
		t.Fatalf("expected error when starting undefined state machine")
	}
}
