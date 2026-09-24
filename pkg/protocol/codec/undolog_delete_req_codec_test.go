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

package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
)

func TestUndoLogDeleteRequestCodec_Encode(t *testing.T) {
	c := &UndoLogDeleteRequestCodec{}

	tests := []struct {
		name string
		req  message.UndoLogDeleteRequest
	}{
		{
			name: "normal case with resource id",
			req: message.UndoLogDeleteRequest{
				ResourceId: "jdbc:mysql://localhost:3306/seata",
				SaveDays:   7,
				BranchType: branch.BranchTypeAT,
			},
		},
		{
			name: "empty resource id",
			req: message.UndoLogDeleteRequest{
				ResourceId: "",
				SaveDays:   30,
				BranchType: branch.BranchTypeAT,
			},
		},
		{
			name: "max save days",
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource",
				SaveDays:   32767, // max int16
				BranchType: branch.BranchTypeAT,
			},
		},
		{
			name: "min save days",
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource",
				SaveDays:   -32768, // min int16
				BranchType: branch.BranchTypeAT,
			},
		},
		{
			name: "zero save days",
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource",
				SaveDays:   0,
				BranchType: branch.BranchTypeAT,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := c.Encode(tt.req)
			assert.NotNil(t, encoded)
			assert.Greater(t, len(encoded), 0, "encoded data should not be empty")
		})
	}
}

func TestUndoLogDeleteRequestCodec_Decode(t *testing.T) {
	c := &UndoLogDeleteRequestCodec{}

	tests := []struct {
		name     string
		req      message.UndoLogDeleteRequest
		validate func(t *testing.T, decoded message.UndoLogDeleteRequest)
	}{
		{
			name: "decode normal request",
			req: message.UndoLogDeleteRequest{
				ResourceId: "jdbc:mysql://localhost:3306/seata",
				SaveDays:   7,
				BranchType: branch.BranchTypeAT,
			},
			validate: func(t *testing.T, decoded message.UndoLogDeleteRequest) {
				assert.Equal(t, "jdbc:mysql://localhost:3306/seata", decoded.ResourceId)
				assert.Equal(t, int16(7), decoded.SaveDays)
				assert.Equal(t, branch.BranchTypeAT, decoded.BranchType)
			},
		},
		{
			name: "decode request with empty resource id",
			req: message.UndoLogDeleteRequest{
				ResourceId: "",
				SaveDays:   15,
				BranchType: branch.BranchTypeAT,
			},
			validate: func(t *testing.T, decoded message.UndoLogDeleteRequest) {
				assert.Equal(t, "", decoded.ResourceId)
				assert.Equal(t, int16(15), decoded.SaveDays)
				assert.Equal(t, branch.BranchTypeAT, decoded.BranchType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := c.Encode(tt.req)
			assert.NotNil(t, encoded)

			decoded := c.Decode(encoded)
			assert.NotNil(t, decoded)

			decodedReq, ok := decoded.(message.UndoLogDeleteRequest)
			assert.True(t, ok, "decoded result should be UndoLogDeleteRequest type")

			tt.validate(t, decodedReq)
		})
	}
}

func TestUndoLogDeleteRequestCodec_EncodeDecode(t *testing.T) {
	c := &UndoLogDeleteRequestCodec{}

	tests := []struct {
		name string
		req  message.UndoLogDeleteRequest
	}{
		{
			name: "round trip - normal case",
			req: message.UndoLogDeleteRequest{
				ResourceId: "jdbc:mysql://127.0.0.1:3306/test_db",
				SaveDays:   10,
				BranchType: branch.BranchTypeAT,
			},
		},
		{
			name: "round trip - empty resource id",
			req: message.UndoLogDeleteRequest{
				ResourceId: "",
				SaveDays:   5,
				BranchType: branch.BranchTypeAT,
			},
		},
		{
			name: "round trip - negative save days",
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource",
				SaveDays:   -1,
				BranchType: branch.BranchTypeAT,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := c.Encode(tt.req)
			assert.NotNil(t, encoded)

			decoded := c.Decode(encoded)
			assert.NotNil(t, decoded)

			decodedReq, ok := decoded.(message.UndoLogDeleteRequest)
			assert.True(t, ok)

			assert.Equal(t, tt.req.ResourceId, decodedReq.ResourceId, "ResourceId should match")
			assert.Equal(t, tt.req.SaveDays, decodedReq.SaveDays, "SaveDays should match")
			assert.Equal(t, tt.req.BranchType, decodedReq.BranchType, "BranchType should match")
		})
	}
}

func TestUndoLogDeleteRequestCodec_GetMessageType(t *testing.T) {
	c := &UndoLogDeleteRequestCodec{}
	assert.Equal(t, message.MessageTypeRmDeleteUndolog, c.GetMessageType(),
		"message type should be MessageTypeRmDeleteUndolog (111)")
}

// Byte vectors hand-derived from Java UndoLogDeleteRequestCodec#encode:
// branchType ordinal (1 byte) + resourceId (uint16-BE length + UTF-8) + saveDays (int16-BE).
func TestUndoLogDeleteRequestCodec_JavaWireFormat(t *testing.T) {
	c := &UndoLogDeleteRequestCodec{}

	tests := []struct {
		name string
		in   []byte
		want message.UndoLogDeleteRequest
	}{
		{
			name: "AT mysql resource 7 days",
			in: []byte{
				0x00,
				0x00, 0x21,
				'j', 'd', 'b', 'c', ':', 'm', 'y', 's', 'q', 'l', ':', '/', '/',
				'1', '2', '7', '.', '0', '.', '0', '.', '1', ':', '3', '3', '0', '6',
				'/', 's', 'e', 'a', 't', 'a',
				0x00, 0x07,
			},
			want: message.UndoLogDeleteRequest{
				BranchType: branch.BranchTypeAT,
				ResourceId: "jdbc:mysql://127.0.0.1:3306/seata",
				SaveDays:   7,
			},
		},
		{
			name: "AT empty resource id 30 days",
			in:   []byte{0x00, 0x00, 0x00, 0x00, 0x1e},
			want: message.UndoLogDeleteRequest{
				BranchType: branch.BranchTypeAT,
				ResourceId: "",
				SaveDays:   30,
			},
		},
		{
			name: "AT negative save days two complement",
			in:   []byte{0x00, 0x00, 0x02, 'p', 'g', 0xff, 0xff},
			want: message.UndoLogDeleteRequest{
				BranchType: branch.BranchTypeAT,
				ResourceId: "pg",
				SaveDays:   -1,
			},
		},
		{
			name: "XA ordinal 3 14 days",
			in:   []byte{0x03, 0x00, 0x02, 'x', 'a', 0x00, 0x0e},
			want: message.UndoLogDeleteRequest{
				BranchType: branch.BranchTypeXA,
				ResourceId: "xa",
				SaveDays:   14,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, ok := c.Decode(tt.in).(message.UndoLogDeleteRequest)
			assert.True(t, ok, "decoded result should be UndoLogDeleteRequest type")
			assert.Equal(t, tt.want, req)
			assert.Equal(t, tt.in, c.Encode(tt.want), "encode must reproduce the Java bytes exactly")
		})
	}
}

func TestUndoLogDeleteRequestCodec_DecodeMalformed(t *testing.T) {
	c := &UndoLogDeleteRequestCodec{}

	tests := []struct {
		name string
		in   []byte
	}{
		{name: "empty input", in: []byte{}},
		{name: "branch type only", in: []byte{0x00}},
		{name: "missing save days", in: []byte{0x00, 0x00, 0x01, 'a'}},
		{name: "truncated resourceId (declares 10 bytes, only 3 provided)", in: []byte{0x00, 0x00, 0x0A, 'a', 'b', 'c'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Nil(t, c.Decode(tt.in), "malformed input should be rejected with nil")
		})
	}
}

func TestUndoLogDeleteRequestCodec_Integration(t *testing.T) {
	Init()

	cm := GetCodecManager()
	c := cm.GetCodec(CodecTypeSeata, message.MessageTypeRmDeleteUndolog)
	assert.NotNil(t, c, "UndoLogDeleteRequest codec should be registered")

	req := message.UndoLogDeleteRequest{
		ResourceId: "integration-test-resource",
		SaveDays:   20,
		BranchType: branch.BranchTypeAT,
	}

	encoded := cm.Encode(CodecTypeSeata, req)
	assert.NotNil(t, encoded)
	assert.Greater(t, len(encoded), 2, "encoded data should include type code (2 bytes) + body")

	decoded := cm.Decode(CodecTypeSeata, encoded)
	assert.NotNil(t, decoded)

	decodedReq, ok := decoded.(message.UndoLogDeleteRequest)
	assert.True(t, ok)
	assert.Equal(t, req.ResourceId, decodedReq.ResourceId)
	assert.Equal(t, req.SaveDays, decodedReq.SaveDays)
	assert.Equal(t, req.BranchType, decodedReq.BranchType)
}

func BenchmarkUndoLogDeleteRequestCodec_Encode(b *testing.B) {
	c := &UndoLogDeleteRequestCodec{}
	req := message.UndoLogDeleteRequest{
		ResourceId: "jdbc:mysql://localhost:3306/seata",
		SaveDays:   7,
		BranchType: branch.BranchTypeAT,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Encode(req)
	}
}

func BenchmarkUndoLogDeleteRequestCodec_Decode(b *testing.B) {
	c := &UndoLogDeleteRequestCodec{}
	req := message.UndoLogDeleteRequest{
		ResourceId: "jdbc:mysql://localhost:3306/seata",
		SaveDays:   7,
		BranchType: branch.BranchTypeAT,
	}
	encoded := c.Encode(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Decode(encoded)
	}
}
