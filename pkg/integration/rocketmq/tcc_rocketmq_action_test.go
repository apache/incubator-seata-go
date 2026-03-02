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

package rocketmq

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/rm/tcc"
)

func TestTCCRocketMQAction_ParseTCCResource(t *testing.T) {
	action := &TCCRocketMQAction{}

	resource, err := tcc.ParseTCCResource(action)
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	assert.Equal(t, ResourceIDTCCRocketMQ, resource.GetResourceId())
	assert.Equal(t, ResourceIDTCCRocketMQ, action.GetActionName())
}

func TestTCCRocketMQAction_GetActionName(t *testing.T) {
	action := &TCCRocketMQAction{}
	assert.Equal(t, ResourceIDTCCRocketMQ, action.GetActionName())
}
