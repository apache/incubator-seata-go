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
	"bytes"
	"sync"

	"vimagination.zapto.org/byteio"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"
)

type CodecType byte

const (
	CodecTypeSeata    = CodecType(0x1)
	CodecTypeProtobuf = CodecType(0x2)
	CodecTypeKRYO     = CodecType(0x4)
	CodecTypeFST      = CodecType(0x8)
)

type Codec interface {
	Encode(in interface{}) []byte
	Decode(in []byte) interface{}
	GetMessageType() message.MessageType
}

var (
	codecManager     *CodecManager
	onceCodecManager = &sync.Once{}
)

func GetCodecManager() *CodecManager {
	if codecManager == nil {
		onceCodecManager.Do(func() {
			codecManager = &CodecManager{
				codecMap: make(map[CodecType]map[message.MessageType]Codec, 0),
			}
		})
	}
	return codecManager
}

type CodecManager struct {
	mutex    sync.Mutex
	codecMap map[CodecType]map[message.MessageType]Codec
}

func (c *CodecManager) RegisterCodec(codecType CodecType, codec Codec) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	codecTypeMap := c.codecMap[codecType]
	if codecTypeMap == nil {
		codecTypeMap = make(map[message.MessageType]Codec, 0)
		c.codecMap[codecType] = codecTypeMap
	}
	codecTypeMap[codec.GetMessageType()] = codec
}

func (c *CodecManager) GetCodec(codecType CodecType, msgType message.MessageType) Codec {
	if m := c.codecMap[codecType]; m != nil {
		return m[msgType]
	}
	return nil
}

func (c *CodecManager) Decode(codecType CodecType, in []byte) interface{} {
	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	typeCode, _, _ := r.ReadInt16()
	codec := c.GetCodec(codecType, message.MessageType(typeCode))

	if codec == nil {
		log.Errorf("This message type [%v] has no codec to decode", typeCode)
		return nil
	}
	return codec.Decode(in[2:])
}

func (c *CodecManager) Encode(codecType CodecType, in interface{}) []byte {
	result := make([]byte, 0)
	msg := in.(message.MessageTypeAware)
	typeCode := msg.GetTypeCode()

	codec := c.GetCodec(codecType, typeCode)
	if codec == nil {
		log.Errorf("This message type [%v] has no codec to encode", typeCode)
		return nil
	}

	body := codec.Encode(in)
	typeC := uint16(typeCode)
	result = append(result, []byte{byte(typeC >> 8), byte(typeC)}...)
	result = append(result, body...)

	return result
}

func Init() {
	// Global
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalReportResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalBeginRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalBeginResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalCommitRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalCommitResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalLockQueryRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalLockQueryResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalRollbackRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalRollbackResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalStatusRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &GlobalStatusResponseCodec{})

	// Branch
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchCommitRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchCommitResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchRegisterRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchRegisterResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchReportRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchRollbackRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchRollbackResponseCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &BranchReportResponseCodec{})

	// RM
	GetCodecManager().RegisterCodec(CodecTypeSeata, &RegisterRMRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &RegisterRMResponseCodec{})

	// TM
	GetCodecManager().RegisterCodec(CodecTypeSeata, &RegisterTMRequestCodec{})
	GetCodecManager().RegisterCodec(CodecTypeSeata, &RegisterTMResponseCodec{})
}
