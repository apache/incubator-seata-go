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
	"bytes"
	"fmt"
	"github.com/seata/seata-go/pkg/saga/statemachine"
	"io"
	"os"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
)

// ConfigParser is a general configuration parser interface, used to agree on the implementation of different types of parsers
type ConfigParser interface {
	Parse(configContent []byte) (*statemachine.StateMachineObject, error)
}

type JSONConfigParser struct{}

func NewJSONConfigParser() *JSONConfigParser {
	return &JSONConfigParser{}
}

func (p *JSONConfigParser) Parse(configContent []byte) (*statemachine.StateMachineObject, error) {
	if configContent == nil || len(configContent) == 0 {
		return nil, fmt.Errorf("empty JSON config content")
	}

	k := koanf.New(".")
	if err := k.Load(rawbytes.Provider(configContent), json.Parser()); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config content: %w", err)
	}

	var stateMachineObject statemachine.StateMachineObject
	if err := k.Unmarshal("", &stateMachineObject); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON config to struct: %w", err)
	}

	return &stateMachineObject, nil
}

type YAMLConfigParser struct{}

func NewYAMLConfigParser() *YAMLConfigParser {
	return &YAMLConfigParser{}
}

func (p *YAMLConfigParser) Parse(configContent []byte) (*statemachine.StateMachineObject, error) {
	if configContent == nil || len(configContent) == 0 {
		return nil, fmt.Errorf("empty YAML config content")
	}

	k := koanf.New(".")
	if err := k.Load(rawbytes.Provider(configContent), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config content: %w", err)
	}

	var stateMachineObject statemachine.StateMachineObject
	if err := k.Unmarshal("", &stateMachineObject); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML config to struct: %w", err)
	}

	return &stateMachineObject, nil
}

type StateMachineConfigParser struct{}

func NewStateMachineConfigParser() *StateMachineConfigParser {
	return &StateMachineConfigParser{}
}

func (p *StateMachineConfigParser) CheckConfigFile(filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file %s does not exist: %w", filePath, err)
	}
	if err != nil {
		return fmt.Errorf("failed to access config file %s: %w", filePath, err)
	}
	return nil
}

func (p *StateMachineConfigParser) ReadConfigFile(configFilePath string) ([]byte, error) {
	file, _ := os.Open(configFilePath)
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var buf bytes.Buffer
	_, err := io.Copy(&buf, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFilePath, err)
	}

	return buf.Bytes(), nil
}

func (p *StateMachineConfigParser) getParser(content []byte) (ConfigParser, error) {
	k := koanf.New(".")
	if err := k.Load(rawbytes.Provider(content), json.Parser()); err == nil {
		return NewJSONConfigParser(), nil
	}

	k = koanf.New(".")
	if err := k.Load(rawbytes.Provider(content), yaml.Parser()); err == nil {
		return NewYAMLConfigParser(), nil
	}

	return nil, fmt.Errorf("unsupported config file format")
}

func (p *StateMachineConfigParser) Parse(content []byte) (*statemachine.StateMachineObject, error) {
	parser, err := p.getParser(content)
	if err != nil {
		return nil, err
	}

	return parser.Parse(content)
}
