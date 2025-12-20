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

package parser

import (
	"os"
	"testing"
)

func readFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func TestParseChoice(t *testing.T) {
	parser := NewJSONStateMachineParser()

	tests := []struct {
		name           string
		configFilePath string
	}{
		{
			name:           "JSON Simple: StateLang With Choice",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.json",
		},
		{
			name:           "YAML Simple: StateLang With Choice",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := readFileContent(tt.configFilePath)
			if err != nil {
				t.Error("read file fail: " + err.Error())
				return
			}
			_, err = parser.Parse(content)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
		})
	}
}

func TestParseServiceTaskForSimpleStateMachine(t *testing.T) {
	parser := NewJSONStateMachineParser()

	tests := []struct {
		name           string
		configFilePath string
	}{
		{
			name:           "JSON Simple: StateMachine",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.json",
		},
		{
			name:           "YAML Simple: StateMachine",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := readFileContent(tt.configFilePath)
			if err != nil {
				t.Error("read file fail: " + err.Error())
				return
			}
			_, err = parser.Parse(content)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
		})
	}
}

func TestParseServiceTaskForNewDesigner(t *testing.T) {
	parser := NewJSONStateMachineParser()

	tests := []struct {
		name           string
		configFilePath string
	}{
		{
			name:           "JSON Simple: StateMachine New Designer",
			configFilePath: "../../../../../testdata/saga/statelang/state_machine_new_designer.json",
		},
		{
			name:           "YAML Simple: StateMachine New Designer",
			configFilePath: "../../../../../testdata/saga/statelang/state_machine_new_designer.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := readFileContent(tt.configFilePath)
			if err != nil {
				t.Error("read file fail: " + err.Error())
				return
			}
			_, err = parser.Parse(content)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
		})
	}
}
