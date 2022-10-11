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
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func Test_StringSliceCSV(t *testing.T) {
	type TestStruct struct {
		CSV StringSliceCSV `yaml:"csv"`
	}

	var testStruct TestStruct
	s := "a,b,c,d"
	assert.Nil(t, testStruct.CSV.Set(s))

	assert.Equal(t, []string{"a", "b", "c", "d"}, []string(testStruct.CSV))
	assert.Equal(t, s, testStruct.CSV.String())

	expected := []byte(`csv: a,b,c,d
`)

	actual, err := yaml.Marshal(testStruct)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)

	var testStruct2 TestStruct

	err = yaml.Unmarshal(expected, &testStruct2)
	assert.Nil(t, err)
	assert.Equal(t, testStruct, testStruct2)
}
