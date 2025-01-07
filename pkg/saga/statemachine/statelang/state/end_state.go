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

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type EndState interface {
	statelang.State
}

type SucceedEndState interface {
	EndState
}

type SucceedEndStateImpl struct {
	*statelang.BaseState
}

func NewSucceedEndStateImpl() *SucceedEndStateImpl {
	s := &SucceedEndStateImpl{
		BaseState: statelang.NewBaseState(),
	}
	s.SetType(constant.StateTypeSucceed)
	return s
}

type FailEndState interface {
	EndState

	ErrorCode() string

	Message() string
}

type FailEndStateImpl struct {
	*statelang.BaseState
	errorCode string
	message   string
}

func NewFailEndStateImpl() *FailEndStateImpl {
	s := &FailEndStateImpl{
		BaseState: statelang.NewBaseState(),
	}
	s.SetType(constant.StateTypeFail)
	return s
}

func (f *FailEndStateImpl) ErrorCode() string {
	return f.errorCode
}

func (f *FailEndStateImpl) SetErrorCode(errorCode string) {
	f.errorCode = errorCode
}

func (f *FailEndStateImpl) Message() string {
	return f.message
}

func (f *FailEndStateImpl) SetMessage(message string) {
	f.message = message
}
