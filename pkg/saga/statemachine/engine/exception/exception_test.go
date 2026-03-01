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

package exception

import (
	"errors"
	"testing"

	pkgerr "seata.apache.org/seata-go/v2/pkg/util/errors"
)

func TestIsEngineExecutionException(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantOk  bool
		wantMsg string
	}{
		{
			name:    "EngineExecutionException",
			err:     &EngineExecutionException{SeataError: pkgerr.SeataError{Message: "engine error"}},
			wantOk:  true,
			wantMsg: "engine error",
		},
		{
			name:    "Other error",
			err:     errors.New("some other error"),
			wantOk:  false,
			wantMsg: "",
		},
		{
			name:    "nil error",
			err:     nil,
			wantOk:  false,
			wantMsg: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fie, ok := IsEngineExecutionException(c.err)
			if ok != c.wantOk {
				t.Errorf("expected ok=%v, got %v", c.wantOk, ok)
			}
			if ok && fie.SeataError.Message != c.wantMsg {
				t.Errorf("expected Message=%q, got %q", c.wantMsg, fie.SeataError.Message)
			}
			if !ok && fie != nil {
				t.Errorf("expected fie=nil, got %v", fie)
			}
		})
	}
}
