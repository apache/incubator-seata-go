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

package getty

import (
	"errors"
	"fmt"

	getty "github.com/apache/dubbo-getty"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/bytes"
)

// 0     1     2     3     4     5     6     7     8     9    10     11    12    13    14    15    16
// +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
// |   magic   |Proto |     Full length      |    Head   | Msg |Seria|Compr|     RequestID         |
// |   code    |clVer |    (head+body)       |   Length  |Type |lizer|ess  |                       |
// +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
// |                                                                                               |
// |                                   Head Map [Optional]                                         |
// +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
// |                                                                                               |
// |                                         body                                                  |
// |                                                                                               |
// |                                        ... ...                                                |
// +-----------------------------------------------------------------------------------------------+
// <li>Full Length: include all data </li>
// <li>Head Length: include head data from magic code to head map. </li>
// <li>Body Length: Full Length - Head Length</li>
// </p>
// https://github.com/seata/seata/issues/893

const (
	Seatav1HeaderLength = 16
)

var (
	magics        = []uint8{0xda, 0xda}
	rpcPkgHandler = &RpcPackageHandler{}
)

var (
	ErrNotEnoughStream = errors.New("packet stream is not enough")
	ErrTooLargePackage = errors.New("package length is exceed the getty package's legal maximum length.")
	ErrInvalidPackage  = errors.New("invalid rpc package")
	ErrIllegalMagic    = errors.New("package magic is not right.")
)

type RpcPackageHandler struct{}

type SeataV1PackageHeader struct {
	Magic0       byte
	Magic1       byte
	Version      byte
	TotalLength  uint32
	HeadLength   uint16
	MessageType  message.GettyRequestType
	CodecType    byte
	CompressType byte
	RequestID    uint32
	Meta         map[string]string
	BodyLength   uint32
	Body         interface{}
}

func (p *RpcPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	in := bytes.NewByteBuffer(data)

	header := SeataV1PackageHeader{}
	magic0 := bytes.ReadByte(in)
	magic1 := bytes.ReadByte(in)
	if magic0 != magics[0] || magic1 != magics[1] {
		return nil, 0, fmt.Errorf("codec decode not found magic offset")
	}
	header.Magic0 = magic0
	header.Magic1 = magic1
	header.Version = bytes.ReadByte(in)
	// length of head and body
	header.TotalLength = bytes.ReadUInt32(in)
	header.HeadLength = bytes.ReadUInt16(in)
	header.MessageType = message.GettyRequestType(bytes.ReadByte(in))
	header.CodecType = bytes.ReadByte(in)
	header.CompressType = bytes.ReadByte(in)
	header.RequestID = bytes.ReadUInt32(in)
	headMapLength := header.HeadLength - Seatav1HeaderLength
	header.Meta = decodeHeapMap(in, headMapLength)
	header.BodyLength = header.TotalLength - uint32(header.HeadLength)

	if uint32(len(data)) < header.TotalLength {
		return nil, int(header.TotalLength), nil
	}

	// r := byteio.BigEndianReader{Reader: bytes.NewReader(data)}
	rpcMessage := message.RpcMessage{
		Codec:      header.CodecType,
		ID:         int32(header.RequestID),
		Compressor: header.CompressType,
		Type:       header.MessageType,
		HeadMap:    header.Meta,
	}

	if header.MessageType == message.GettyRequestTypeHeartbeatRequest {
		rpcMessage.Body = message.HeartBeatMessagePing
	} else if header.MessageType == message.GettyRequestTypeHeartbeatResponse {
		rpcMessage.Body = message.HeartBeatMessagePong
	} else {
		if header.BodyLength > 0 {
			msg := codec.GetCodecManager().Decode(codec.CodecType(header.CodecType), data[header.HeadLength:])
			rpcMessage.Body = msg
		}
	}

	return rpcMessage, int(header.TotalLength), nil
}

// Write write rpc message to binary data
func (p *RpcPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	msg, ok := pkg.(message.RpcMessage)
	if !ok {
		return nil, ErrInvalidPackage
	}

	totalLength := message.V1HeadLength
	headLength := message.V1HeadLength

	var headMapBytes []byte
	if msg.HeadMap != nil && len(msg.HeadMap) > 0 {
		hb, headMapLength := encodeHeapMap(msg.HeadMap)
		headMapBytes = hb
		headLength += headMapLength
		totalLength += headMapLength
	}

	var bodyBytes []byte
	if msg.Type != message.GettyRequestTypeHeartbeatRequest &&
		msg.Type != message.GettyRequestTypeHeartbeatResponse {
		bodyBytes = codec.GetCodecManager().Encode(codec.CodecType(msg.Codec), msg.Body)
		totalLength += len(bodyBytes)
	}

	buf := bytes.NewByteBuffer([]byte{})
	buf.WriteByte(message.MagicCodeBytes[0])
	buf.WriteByte(message.MagicCodeBytes[1])
	buf.WriteByte(message.VERSION)
	buf.WriteUint32(uint32(totalLength))
	buf.WriteUint16(uint16(headLength))
	buf.WriteByte(byte(msg.Type))
	buf.WriteByte(msg.Codec)
	buf.WriteByte(msg.Compressor)
	buf.WriteUint32(uint32(msg.ID))
	buf.Write(headMapBytes)
	buf.Write(bodyBytes)

	return buf.Bytes(), nil
}

func encodeHeapMap(data map[string]string) ([]byte, int) {
	buf := bytes.NewByteBuffer([]byte{})
	for k, v := range data {
		if k == "" {
			buf.WriteUint16(uint16(0))
		} else {
			buf.WriteUint16(uint16(len(k)))
			buf.WriteString(k)
		}

		if v == "" {
			buf.WriteUint16(uint16(0))
		} else {
			buf.WriteUint16(uint16(len(v)))
			buf.WriteString(v)
		}
	}
	res := buf.Bytes()
	return res, len(res)
}

func decodeHeapMap(in *bytes.ByteBuffer, length uint16) map[string]string {
	res := make(map[string]string, 0)
	if length == 0 {
		return res
	}

	readedLength := uint16(0)
	for readedLength < length {
		var key, value string
		keyLength := bytes.ReadUInt16(in)
		if keyLength == 0 {
			key = ""
		} else {
			keyBytes := make([]byte, keyLength)
			in.Read(keyBytes)
			key = string(keyBytes)
		}

		valueLength := bytes.ReadUInt16(in)
		if valueLength == 0 {
			key = ""
		} else {
			valueBytes := make([]byte, valueLength)
			in.Read(valueBytes)
			value = string(valueBytes)
		}

		res[key] = value
		readedLength += 4 + keyLength + valueLength
		fmt.Sprintln("done")
	}
	return res
}
