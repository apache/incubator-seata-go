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

package reflectx

import (
	"testing"
)

func TestGetElemDataValue(t *testing.T) {
	var aa = 10
	var bb = "name"
	var cc bool

	tests := []struct {
		name string
		args interface{}
		want interface{}
	}{
		{name: "test1", args: aa, want: aa},
		{name: "test2", args: &aa, want: aa},
		{name: "test3", args: bb, want: bb},
		{name: "test4", args: &bb, want: bb},
		{name: "test5", args: cc, want: cc},
		{name: "test6", args: &cc, want: cc},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetElemDataValue(tt.args); got != tt.want {
				t.Errorf("GetElemDataValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
