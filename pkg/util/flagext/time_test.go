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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestTimeYAML(t *testing.T) {
	{
		type TestStruct struct {
			T Time `yaml:"time"`
		}

		var testStruct TestStruct
		require.NoError(t, testStruct.T.Set("2020-10-20"))

		marshaled, err := yaml.Marshal(testStruct)
		require.NoError(t, err)

		expected := `time: "2020-10-20T00:00:00Z"` + "\n"
		assert.Equal(t, expected, string(marshaled))

		var actualStruct TestStruct
		err = yaml.Unmarshal([]byte(expected), &actualStruct)
		require.NoError(t, err)
		assert.Equal(t, testStruct, actualStruct)
	}

	{
		type TestStruct struct {
			T *Time `yaml:"time"`
		}

		var testStruct TestStruct
		testStruct.T = &Time{}
		require.NoError(t, testStruct.T.Set("2020-10-20"))

		marshaled, err := yaml.Marshal(testStruct)
		require.NoError(t, err)

		expected := `time: "2020-10-20T00:00:00Z"` + "\n"
		assert.Equal(t, expected, string(marshaled))

		var actualStruct TestStruct
		err = yaml.Unmarshal([]byte(expected), &actualStruct)
		require.NoError(t, err)
		assert.Equal(t, testStruct, actualStruct)
	}
}

func TestTimeFormats(t *testing.T) {
	ts := &Time{}
	require.NoError(t, ts.Set("0"))
	require.True(t, time.Time(*ts).IsZero())
	require.Equal(t, "0", ts.String())

	require.NoError(t, ts.Set("2020-10-05"))
	require.Equal(t, mustParseTime(t, "2006-01-02", "2020-10-05").Format(time.RFC3339), ts.String())

	require.NoError(t, ts.Set("2020-10-05T13:27"))
	require.Equal(t, mustParseTime(t, "2006-01-02T15:04", "2020-10-05T13:27").Format(time.RFC3339), ts.String())

	require.NoError(t, ts.Set("2020-10-05T13:27:00Z"))
	require.Equal(t, mustParseTime(t, "2006-01-02T15:04:05Z", "2020-10-05T13:27:00Z").Format(time.RFC3339), ts.String())

	require.NoError(t, ts.Set("2020-10-05T13:27:00+02:00"))
	require.Equal(t, mustParseTime(t, "2006-01-02T15:04:05Z07:00", "2020-10-05T13:27:00+02:00").Format(time.RFC3339), ts.String())
}

func mustParseTime(t *testing.T, f, s string) time.Time {
	ts, err := time.Parse(f, s)
	if err != nil {
		t.Fatalf("failed to parse %q: %v", s, err)
	}
	return ts
}
