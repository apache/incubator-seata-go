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

func TestProtobufGetName(t *testing.T) {
	assert.Equal(t, "protobuf", (&ProtobufParser{}).GetName())
}

func TestProtobufDefaultContext(t *testing.T) {
	assert.Equal(t, []byte{}, (&ProtobufParser{}).GetDefaultContent())
}

func TestProtobufEncodeDecode(t *testing.T) {
	TestCases := []struct {
		CaseName  string
		UndoLog   *undo.BranchUndoLog
		ExpectErr bool
	}{
		{
			CaseName: "pass",
			UndoLog: &undo.BranchUndoLog{
				Xid:      "123456",
				BranchID: 123456,
				Logs:     []undo.SQLUndoLog{},
			},
			ExpectErr: false,
		},
	}

	for _, Case := range TestCases {
		t.Run(Case.CaseName, func(t *testing.T) {
			parser := &ProtobufParser{}

			encodedBytes, err := parser.Encode(Case.UndoLog)
			if Case.ExpectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)

			decodedUndoLog, err := parser.Decode(encodedBytes)
			if Case.ExpectErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)

			assert.Equal(t, Case.UndoLog.Xid, decodedUndoLog.Xid)
			assert.Equal(t, Case.UndoLog.BranchID, decodedUndoLog.BranchID)
			assert.Equal(t, len(Case.UndoLog.Logs), len(decodedUndoLog.Logs))
		})
	}
}

func TestConvertInterfaceToAnyAndBack(t *testing.T) {
	originalValue := map[string]interface{}{
		"key1": "value1",
		"key2": float64(123), // Use float64 to match JSON default behavior
		"key3": true,
	}

	anyValue, err := convertInterfaceToAny(originalValue)
	assert.NoError(t, err, "convertInterfaceToAny should not return an error")

	convertedValue, err := convertAnyToInterface(anyValue)
	assert.NoError(t, err, "convertAnyToInterface should not return an error")

	assert.Equal(t, originalValue, convertedValue, "The converted value should match the original")
}
