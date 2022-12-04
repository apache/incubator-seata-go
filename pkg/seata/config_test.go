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

package seata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadPath(t *testing.T) {
	c := LoadPath("../../testdata/conf/seatago.yml")
	assert.NotNil(t, c)
	assert.NotNil(t, c.TCCConfig)
	assert.NotNil(t, c.TCCConfig.FenceConfig)

	assert.Equal(t, "tcc_fence_log_test", c.TCCConfig.FenceConfig.LogTableName)
	assert.Equal(t, time.Second*60, c.TCCConfig.FenceConfig.CleanPeriod)
}
