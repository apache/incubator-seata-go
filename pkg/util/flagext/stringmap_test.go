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

package flagext

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StringMap(t *testing.T) {
	type TestStruct struct {
		Val StringMap `json:"val"`
	}

	var testStruct TestStruct
	s := `{"aaa":"bb"}`
	assert.Nil(t, testStruct.Val.Set(s))
	assert.Equal(t, s, testStruct.Val.String())

	expected := []byte(`{"val":{"aaa":"bb"}}`)
	actual, err := json.Marshal(testStruct)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)

	var testStruct2 TestStruct

	err = json.Unmarshal(expected, &testStruct2)
	assert.Nil(t, err)
	assert.Equal(t, testStruct, testStruct2)

}
