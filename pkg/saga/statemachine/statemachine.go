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

package statemachine

import (
	"flag"
)

type StateMachineObject struct {
	Name                        string                 `json:"Name" yaml:"Name"`
	Comment                     string                 `json:"Comment" yaml:"Comment"`
	Version                     string                 `json:"Version" yaml:"Version"`
	StartState                  string                 `json:"StartState" yaml:"StartState"`
	RecoverStrategy             string                 `json:"RecoverStrategy" yaml:"RecoverStrategy"`
	Persist                     bool                   `json:"IsPersist" yaml:"IsPersist"`
	RetryPersistModeUpdate      bool                   `json:"IsRetryPersistModeUpdate" yaml:"IsRetryPersistModeUpdate"`
	CompensatePersistModeUpdate bool                   `json:"IsCompensatePersistModeUpdate" yaml:"IsCompensatePersistModeUpdate"`
	Type                        string                 `json:"Type" yaml:"Type"`
	States                      map[string]interface{} `json:"States" yaml:"States"`
}

func (smo *StateMachineObject) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&smo.Name, prefix+".name", "", "State machine name.")
	f.StringVar(&smo.Comment, prefix+".comment", "", "State machine comment.")
	f.StringVar(&smo.Version, prefix+".version", "1.0", "State machine version.")
	f.StringVar(&smo.StartState, prefix+".start-state", "", "State machine start state.")
	f.StringVar(&smo.RecoverStrategy, prefix+".recover-strategy", "", "State machine recovery strategy.")
	f.BoolVar(&smo.Persist, prefix+".persist", false, "Whether to persist state machine.")
	f.BoolVar(&smo.RetryPersistModeUpdate, prefix+".retry-persist-mode-update", false, "Whether to use update mode for retry persistence.")
	f.BoolVar(&smo.CompensatePersistModeUpdate, prefix+".compensate-persist-mode-update", false, "Whether to use update mode for compensate persistence.")
	f.StringVar(&smo.Type, prefix+".type", "", "State machine type.")
}
