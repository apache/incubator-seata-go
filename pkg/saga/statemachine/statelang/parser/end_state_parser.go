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
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type SucceedEndStateParser struct {
	*BaseStateParser
}

func NewSucceedEndStateParser() *SucceedEndStateParser {
	return &SucceedEndStateParser{
		&BaseStateParser{},
	}
}

func (s SucceedEndStateParser) StateType() string {
	return constant.StateTypeSucceed
}

func (s SucceedEndStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	succeedEndStateImpl := state.NewSucceedEndStateImpl()
	err := s.ParseBaseAttributes(stateName, succeedEndStateImpl, stateMap)
	if err != nil {
		return nil, err
	}

	return succeedEndStateImpl, nil
}

type FailEndStateParser struct {
	*BaseStateParser
}

func NewFailEndStateParser() *FailEndStateParser {
	return &FailEndStateParser{
		&BaseStateParser{},
	}
}

func (f FailEndStateParser) StateType() string {
	return constant.StateTypeFail
}

func (f FailEndStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	failEndStateImpl := state.NewFailEndStateImpl()
	err := f.ParseBaseAttributes(stateName, failEndStateImpl, stateMap)
	if err != nil {
		return nil, err
	}

	errorCode, err := f.GetStringOrDefault(stateName, stateMap, "ErrorCode", "")
	if err != nil {
		return nil, err
	}
	failEndStateImpl.SetErrorCode(errorCode)

	message, err := f.GetStringOrDefault(stateName, stateMap, "Message", "")
	if err != nil {
		return nil, err
	}
	failEndStateImpl.SetMessage(message)
	return failEndStateImpl, nil
}
