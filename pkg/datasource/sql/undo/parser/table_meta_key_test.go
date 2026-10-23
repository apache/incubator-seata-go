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

	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestTableMetaKeyRoundTrip(t *testing.T) {
	key := &types.TableMetaKey{DBName: "tenant_a", Schema: "Sales", TableName: "Orders"}
	original := &undo.BranchUndoLog{Logs: []undo.SQLUndoLog{{TableName: "Orders", TableMetaKey: key}}}
	for _, parser := range []struct {
		name  string
		codec interface {
			Encode(*undo.BranchUndoLog) ([]byte, error)
			Decode([]byte) (*undo.BranchUndoLog, error)
		}
	}{
		{name: "json", codec: &JsonParser{}},
		{name: "protobuf", codec: &ProtobufParser{}},
	} {
		t.Run(parser.name, func(t *testing.T) {
			data, err := parser.codec.Encode(original)
			require.NoError(t, err)
			decoded, err := parser.codec.Decode(data)
			require.NoError(t, err)
			require.Equal(t, key, decoded.Logs[0].TableMetaKey)
		})
	}
}

func TestLegacyUndoLogWithoutTableMetaKey(t *testing.T) {
	for _, parser := range []struct {
		name  string
		codec interface {
			Encode(*undo.BranchUndoLog) ([]byte, error)
			Decode([]byte) (*undo.BranchUndoLog, error)
		}
	}{
		{name: "json", codec: &JsonParser{}},
		{name: "protobuf", codec: &ProtobufParser{}},
	} {
		t.Run(parser.name, func(t *testing.T) {
			data, err := parser.codec.Encode(&undo.BranchUndoLog{Logs: []undo.SQLUndoLog{{TableName: "orders"}}})
			require.NoError(t, err)
			decoded, err := parser.codec.Decode(data)
			require.NoError(t, err)
			require.Nil(t, decoded.Logs[0].TableMetaKey)
		})
	}
}
