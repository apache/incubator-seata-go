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
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
)

func TestUndoLogDeleteRequestCodec_Encode(t *testing.T) {
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
				BranchType: branch.BranchTypeTCC,
			},
		},
		{
			name: "max save days",
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource",
				SaveDays:   32767, // max int16
				BranchType: branch.BranchTypeXA,
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

	codec := &UndoLogDeleteRequestCodec{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := codec.Encode(tt.req)
			assert.NotNil(t, encoded)
			assert.Greater(t, len(encoded), 0, "encoded data should not be empty")
		})
	}
}

func TestUndoLogDeleteRequestCodec_Decode(t *testing.T) {
	codec := &UndoLogDeleteRequestCodec{}

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
			name: "decode request with TCC branch type",
			req: message.UndoLogDeleteRequest{
				ResourceId: "tcc-resource",
				SaveDays:   15,
				BranchType: branch.BranchTypeTCC,
			},
			validate: func(t *testing.T, decoded message.UndoLogDeleteRequest) {
				assert.Equal(t, "tcc-resource", decoded.ResourceId)
				assert.Equal(t, int16(15), decoded.SaveDays)
				assert.Equal(t, branch.BranchTypeTCC, decoded.BranchType)
			},
		},
		{
			name: "decode request with XA branch type",
			req: message.UndoLogDeleteRequest{
				ResourceId: "xa-resource",
				SaveDays:   30,
				BranchType: branch.BranchTypeXA,
			},
			validate: func(t *testing.T, decoded message.UndoLogDeleteRequest) {
				assert.Equal(t, "xa-resource", decoded.ResourceId)
				assert.Equal(t, int16(30), decoded.SaveDays)
				assert.Equal(t, branch.BranchTypeXA, decoded.BranchType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode first
			encoded := codec.Encode(tt.req)
			assert.NotNil(t, encoded)

			// Then decode
			decoded := codec.Decode(encoded)
			assert.NotNil(t, decoded)

			decodedReq, ok := decoded.(message.UndoLogDeleteRequest)
			assert.True(t, ok, "decoded result should be UndoLogDeleteRequest type")

			// Validate
			tt.validate(t, decodedReq)
		})
	}
}

func TestUndoLogDeleteRequestCodec_EncodeDecode(t *testing.T) {
	codec := &UndoLogDeleteRequestCodec{}

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
				BranchType: branch.BranchTypeTCC,
			},
		},
		{
			name: "round trip - negative save days",
			req: message.UndoLogDeleteRequest{
				ResourceId: "test-resource",
				SaveDays:   -1,
				BranchType: branch.BranchTypeXA,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := codec.Encode(tt.req)
			assert.NotNil(t, encoded)

			// Decode
			decoded := codec.Decode(encoded)
			assert.NotNil(t, decoded)

			decodedReq, ok := decoded.(message.UndoLogDeleteRequest)
			assert.True(t, ok)

			// Verify all fields match
			assert.Equal(t, tt.req.ResourceId, decodedReq.ResourceId, "ResourceId should match")
			assert.Equal(t, tt.req.SaveDays, decodedReq.SaveDays, "SaveDays should match")
			assert.Equal(t, tt.req.BranchType, decodedReq.BranchType, "BranchType should match")
		})
	}
}

func TestUndoLogDeleteRequestCodec_GetMessageType(t *testing.T) {
	codec := &UndoLogDeleteRequestCodec{}
	msgType := codec.GetMessageType()

	assert.Equal(t, message.MessageTypeRmDeleteUndolog, msgType,
		"message type should be MessageTypeRmDeleteUndolog (111)")
}

func TestUndoLogDeleteRequestCodec_Integration(t *testing.T) {
	// Test integration with codec manager
	Init() // Initialize all codecs

	cm := GetCodecManager()
	codec := cm.GetCodec(CodecTypeSeata, message.MessageTypeRmDeleteUndolog)

	assert.NotNil(t, codec, "UndoLogDeleteRequest codec should be registered")

	// Test encoding through codec manager
	req := message.UndoLogDeleteRequest{
		ResourceId: "integration-test-resource",
		SaveDays:   20,
		BranchType: branch.BranchTypeAT,
	}

	encoded := cm.Encode(CodecTypeSeata, req)
	assert.NotNil(t, encoded)
	assert.Greater(t, len(encoded), 2, "encoded data should include type code (2 bytes) + body")

	// Test decoding through codec manager
	decoded := cm.Decode(CodecTypeSeata, encoded)
	assert.NotNil(t, decoded)

	decodedReq, ok := decoded.(message.UndoLogDeleteRequest)
	assert.True(t, ok)
	assert.Equal(t, req.ResourceId, decodedReq.ResourceId)
	assert.Equal(t, req.SaveDays, decodedReq.SaveDays)
	assert.Equal(t, req.BranchType, decodedReq.BranchType)
}

func BenchmarkUndoLogDeleteRequestCodec_Encode(b *testing.B) {
	codec := &UndoLogDeleteRequestCodec{}
	req := message.UndoLogDeleteRequest{
		ResourceId: "jdbc:mysql://localhost:3306/seata",
		SaveDays:   7,
		BranchType: branch.BranchTypeAT,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = codec.Encode(req)
	}
}

func BenchmarkUndoLogDeleteRequestCodec_Decode(b *testing.B) {
	codec := &UndoLogDeleteRequestCodec{}
	req := message.UndoLogDeleteRequest{
		ResourceId: "jdbc:mysql://localhost:3306/seata",
		SaveDays:   7,
		BranchType: branch.BranchTypeAT,
	}
	encoded := codec.Encode(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = codec.Decode(encoded)
	}
}
