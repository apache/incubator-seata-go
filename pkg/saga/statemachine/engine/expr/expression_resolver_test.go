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

package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExpressionStruct(t *testing.T) {
	tests := []struct {
		expressionStr string
		expected      *ExpressionStruct
		expectError   bool
	}{
		{
			expressionStr: "$type{content}",
			expected: &ExpressionStruct{
				typeStart: 0,
				typeEnd:   5,
				typ:       "type",
				end:       13,
				content:   "content",
			},
			expectError: false,
		},
		{
			expressionStr: "$type.content",
			expected: &ExpressionStruct{
				typeStart: 0,
				typeEnd:   5,
				typ:       "type",
				end:       13,
				content:   "content",
			},
			expectError: false,
		},
		{
			expressionStr: "invalid expression",
			expected:      nil,
			expectError:   true,
		},
	}
	for _, test := range tests {
		result, err := parseExpressionStruct(test.expressionStr)
		if test.expectError {
			assert.Error(t, err, "Expected an error for input '%s'", test.expressionStr)
		} else {
			assert.NoError(t, err, "Did not expect an error for input '%s'", test.expressionStr)
			assert.NotNil(t, result, "Expected a non-nil result for input '%s'", test.expressionStr)
			assert.Equal(t, *test.expected, *result, "Expected result %+v, got %+v for input '%s'", test.expected, result, test.expressionStr)
		}
	}
}
