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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

type mockUndoLogParser struct {
	name           string
	defaultContent []byte
	shouldError    bool
}

func (m *mockUndoLogParser) GetName() string {
	return m.name
}

func (m *mockUndoLogParser) GetDefaultContent() []byte {
	return m.defaultContent
}

func (m *mockUndoLogParser) Encode(branchUndoLog *undo.BranchUndoLog) ([]byte, error) {
	if m.shouldError {
		return nil, errors.New("mock encode error")
	}
	return []byte("mock-encoded-data"), nil
}

func (m *mockUndoLogParser) Decode(bytes []byte) (*undo.BranchUndoLog, error) {
	if m.shouldError {
		return nil, errors.New("mock decode error")
	}
	return &undo.BranchUndoLog{
		Xid:      "mock-xid",
		BranchID: uint64(12345),
		Logs:     []undo.SQLUndoLog{},
	}, nil
}

func TestUndoLogParserInterface(t *testing.T) {
	tests := []struct {
		name            string
		parser          UndoLogParser
		expectedName    string
		expectedContent []byte
		shouldError     bool
	}{
		{
			name: "mock parser successful operations",
			parser: &mockUndoLogParser{
				name:           "mock",
				defaultContent: []byte("{}"),
				shouldError:    false,
			},
			expectedName:    "mock",
			expectedContent: []byte("{}"),
			shouldError:     false,
		},
		{
			name: "mock parser with error",
			parser: &mockUndoLogParser{
				name:           "error-mock",
				defaultContent: []byte(""),
				shouldError:    true,
			},
			expectedName:    "error-mock",
			expectedContent: []byte(""),
			shouldError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, tt.parser.GetName())
			assert.Equal(t, tt.expectedContent, tt.parser.GetDefaultContent())

			branchUndoLog := &undo.BranchUndoLog{
				Xid:      "test-xid",
				BranchID: 123,
				Logs:     []undo.SQLUndoLog{},
			}

			encodedData, err := tt.parser.Encode(branchUndoLog)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, encodedData)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encodedData)
			}

			testData := []byte("test-data")
			decodedLog, err := tt.parser.Decode(testData)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, decodedLog)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, decodedLog)
				assert.Equal(t, "mock-xid", decodedLog.Xid)
				assert.Equal(t, uint64(12345), decodedLog.BranchID)
			}
		})
	}
}

func TestUndoLogParserInterfaceCompliance(t *testing.T) {
	var parser UndoLogParser

	parser = &JsonParser{}
	assert.Equal(t, "json", parser.GetName())
	assert.Equal(t, []byte("{}"), parser.GetDefaultContent())

	parser = &ProtobufParser{}
	assert.Equal(t, "protobuf", parser.GetName())
	assert.Equal(t, []byte{}, parser.GetDefaultContent())
}

func TestUndoLogParserEncodeDecode(t *testing.T) {
	parsers := []UndoLogParser{
		&JsonParser{},
		&ProtobufParser{},
	}

	branchUndoLog := &undo.BranchUndoLog{
		Xid:      "test-xid-123",
		BranchID: 456789,
		Logs:     []undo.SQLUndoLog{},
	}

	for _, parser := range parsers {
		t.Run("test_"+parser.GetName()+"_encode_decode", func(t *testing.T) {
			encoded, err := parser.Encode(branchUndoLog)
			assert.NoError(t, err)
			assert.NotNil(t, encoded)
			assert.True(t, len(encoded) > 0)

			decoded, err := parser.Decode(encoded)
			assert.NoError(t, err)
			assert.NotNil(t, decoded)
			assert.Equal(t, branchUndoLog.Xid, decoded.Xid)
			assert.Equal(t, branchUndoLog.BranchID, decoded.BranchID)
		})
	}
}

func TestUndoLogParserInvalidData(t *testing.T) {
	parsers := []UndoLogParser{
		&JsonParser{},
		&ProtobufParser{},
	}

	invalidData := []byte("invalid-data-that-cannot-be-parsed")

	for _, parser := range parsers {
		t.Run("test_"+parser.GetName()+"_decode_invalid_data", func(t *testing.T) {
			decoded, err := parser.Decode(invalidData)
			assert.Error(t, err)
			if parser.GetName() == "json" {
				assert.Nil(t, decoded)
			} else {
				assert.Nil(t, decoded)
			}
		})
	}
}
