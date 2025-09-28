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

package skills

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Skill interface {
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	Name() string
	Description() string
}

type ExampleSkill struct {
	logger *zap.Logger
}

func NewExampleSkill(logger *zap.Logger) *ExampleSkill {
	return &ExampleSkill{logger: logger}
}

func (s *ExampleSkill) Name() string {
	return "example"
}

func (s *ExampleSkill) Description() string {
	return "An example skill that processes input text"
}

func (s *ExampleSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	input, ok := params["input"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'input' parameter")
	}

	s.logger.Info("Executing example skill", zap.String("input", input))

	result := fmt.Sprintf("Processed: %s (length: %d)", input, len(input))

	return map[string]interface{}{
		"result":   result,
		"original": input,
		"length":   len(input),
	}, nil
}
