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

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestGetName(t *testing.T) {
	assert.Equal(t, "json", (&JsonParser{}).GetName())
}

func TestGetDefaultContext(t *testing.T) {
	assert.Equal(t, []byte("{}"), (&JsonParser{}).GetDefaultContent())
}

func TestEncode(t *testing.T) {
	TestCases := []struct {
		CaseName    string
		UndoLog     *undo.BranchUndoLog
		ExpectErr   bool
		ExpectBytes string
	}{
		{
			CaseName: "pass",
			UndoLog: &undo.BranchUndoLog{
				Xid:      "123456",
				BranchID: 123456,
				Logs:     []undo.SQLUndoLog{},
			},
			ExpectErr:   false,
			ExpectBytes: `{"xid":"123456","branchId":123456,"sqlUndoLogs":[]}`,
		},
	}

	for _, Case := range TestCases {
		t.Run(Case.CaseName, func(t *testing.T) {
			logParser := &JsonParser{}
			bytes, err := logParser.Encode(Case.UndoLog)
			if Case.ExpectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, Case.ExpectBytes, string(bytes))
		})
	}
}

func TestDecode(t *testing.T) {
	TestCases := []struct {
		CaseName      string
		ExpectUndoLog undo.BranchUndoLog
		ExpectErr     bool
		InputBytes    string
	}{
		{
			CaseName: "pass",
			ExpectUndoLog: undo.BranchUndoLog{
				Xid:      "123456",
				BranchID: 123456,
				Logs:     []undo.SQLUndoLog{},
			},
			ExpectErr:  false,
			InputBytes: `{"xid":"123456","branchId":123456,"sqlUndoLogs":[]}`,
		},
	}

	for _, Case := range TestCases {
		t.Run(Case.CaseName, func(t *testing.T) {
			logParser := &JsonParser{}
			undoLog, err := logParser.Decode([]byte(Case.InputBytes))
			if Case.ExpectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, undoLog.Xid, Case.ExpectUndoLog.Xid)
			assert.Equal(t, undoLog.BranchID, Case.ExpectUndoLog.BranchID)
			assert.Equal(t, len(undoLog.Logs), len(Case.ExpectUndoLog.Logs))
		})
	}

}
