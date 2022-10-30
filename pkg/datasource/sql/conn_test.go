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

package sql

import (
	"reflect"
	"testing"

	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/exec/xa"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/util/reflectx"
	"github.com/stretchr/testify/assert"
)

func TestConn_BuildATExecutor(t *testing.T) {
	executor, err := exec.BuildExecutor(types.DBTypeMySQL, types.ATMode, "SELECT * FROM user")

	assert.NoError(t, err)
	_, ok := executor.(*exec.BaseExecutor)
	assert.True(t, ok, "need base executor")
}

func TestConn_BuildXAExecutor(t *testing.T) {
	executor, err := exec.BuildExecutor(types.DBTypeMySQL, types.XAMode, "SELECT * FROM user")

	assert.NoError(t, err)
	val, ok := executor.(*exec.BaseExecutor)
	assert.True(t, ok, "need base executor")

	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName("ex")

	fieldVal := reflectx.GetUnexportedField(field)

	_, ok = fieldVal.(*xa.XAExecutor)
	assert.True(t, ok, "need xa executor")
}
