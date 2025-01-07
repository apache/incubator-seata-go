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

package state

import "github.com/seata/seata-go/pkg/saga/statemachine/statelang"

type ChoiceState interface {
	statelang.State

	Choices() []Choice

	Default() string
}

type Choice interface {
	Expression() string

	SetExpression(expression string)

	Next() string

	SetNext(next string)
}

type ChoiceStateImpl struct {
	*statelang.BaseState
	defaultChoice string   `alias:"Default"`
	choices       []Choice `alias:"Choices"`
}

func NewChoiceStateImpl() *ChoiceStateImpl {
	return &ChoiceStateImpl{
		BaseState: statelang.NewBaseState(),
		choices:   make([]Choice, 0),
	}
}

func (choiceState *ChoiceStateImpl) Default() string {
	return choiceState.defaultChoice
}

func (choiceState *ChoiceStateImpl) Choices() []Choice {
	return choiceState.choices
}

func (choiceState *ChoiceStateImpl) SetDefault(defaultChoice string) {
	choiceState.defaultChoice = defaultChoice
}

func (choiceState *ChoiceStateImpl) SetChoices(choices []Choice) {
	choiceState.choices = choices
}

type ChoiceImpl struct {
	expression string
	next       string
}

func NewChoiceImpl() *ChoiceImpl {
	return &ChoiceImpl{}
}

func (c *ChoiceImpl) Expression() string {
	return c.expression
}

func (c *ChoiceImpl) SetExpression(expression string) {
	c.expression = expression
}

func (c *ChoiceImpl) Next() string {
	return c.next
}

func (c *ChoiceImpl) SetNext(next string) {
	c.next = next
}
