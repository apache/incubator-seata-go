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

func Test_CIDRSliceCSV_YamlMarshaling(t *testing.T) {
	type TestStruct struct {
		CIDRs CIDRSliceCSV `yaml:"cidrs"`
	}

	tests := map[string]struct {
		input    string
		expected []string
	}{
		"should marshal empty config": {
			input:    "cidrs: \"\"\n",
			expected: nil,
		},
		"should marshal single value": {
			input:    "cidrs: demo.wuxian.pro/32\n",
			expected: []string{"demo.wuxian.pro/32"},
		},
		"should marshal multiple comma-separated values": {
			input:    "cidrs: demo.wuxian.pro/32,10.0.10.0/28,fdf8:f53b:82e4::/100,192.168.0.0/20\n",
			expected: []string{"demo.wuxian.pro/32", "10.0.10.0/28", "fdf8:f53b:82e4::/100", "192.168.0.0/20"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Unmarshal.
			actual := TestStruct{}
			err := yaml.Unmarshal([]byte(tc.input), &actual)
			assert.NoError(t, err)

			assert.Len(t, actual.CIDRs, len(tc.expected))
			for idx, cidr := range actual.CIDRs {
				assert.Equal(t, tc.expected[idx], cidr.String())
			}

			// Marshal.
			out, err := yaml.Marshal(actual)
			assert.NoError(t, err)
			assert.Equal(t, tc.input, string(out))
		})
	}
}
