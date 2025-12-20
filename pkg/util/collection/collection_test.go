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

package collection

import (
	"reflect"
	"testing"
)

func TestEncodeDecodeMap(t *testing.T) {
	testCases := []struct {
		name     string
		dataMap  map[string]string
		expected map[string]string
	}{
		{
			name: "test case 1",
			dataMap: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:     "test case 2",
			dataMap:  nil,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeMap(tc.dataMap)
			decoded := DecodeMap(encoded)

			if !reflect.DeepEqual(decoded, tc.expected) {
				t.Errorf("DecodeMap(%v) = %v, expected %v", encoded, decoded, tc.expected)
			}
		})
	}
}

func TestStack(t *testing.T) {
	stack := NewStack()
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)
	stack.Push(4)

	len := stack.Len()
	if len != 4 {
		t.Errorf("stack.Len() failed. Got %d, expected 4.", len)
	}

	value := stack.Peak().(int)
	if value != 4 {
		t.Errorf("stack.Peak() failed. Got %d, expected 4.", value)
	}

	value = stack.Pop().(int)
	if value != 4 {
		t.Errorf("stack.Pop() failed. Got %d, expected 4.", value)
	}
}
