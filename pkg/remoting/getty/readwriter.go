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
	"fmt"

	"github.com/seata/seata-go/pkg/util/bytes"

	getty "github.com/apache/dubbo-getty"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/protocol/codec"
	"github.com/seata/seata-go/pkg/protocol/message"
)

/**
 * <pre>
 * 0     1     2     3     4     5     6     7     8     9    10     11    12    13    14    15    16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |   magic   |Proto |     Full length      |    Head   | Msg |Seria|Compr|     RequestID         |
 * |   code    |clVer |    (head+body)       |   Length  |Type |lizer|ess  |                       |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |                                                                                               |
 * |                                   Head Map [Optional]                                         |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |                                                                                               |
 * |                                         body                                                  |
 * |                                                                                               |
 * |                                        ... ...                                                |
 * +-----------------------------------------------------------------------------------------------+
 * </pre>
 * <p>
 * <li>Full Length: include all data </li>
 * <li>Head Length: include head data from magic code to head map. </li>
 * <li>Body Length: Full Length - Head Length</li>
 * </p>
 * https://github.com/seata/seata/issues/893
 */
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
	err := buf.WriteByte(message.MagicCodeBytes[0])
	if err != nil {
		return nil, err
	}
	err = buf.WriteByte(message.MagicCodeBytes[1])
	if err != nil {
		return nil, err
	}
	err = buf.WriteByte(message.VERSION)
	if err != nil {
		return nil, err
	}
	_, err = buf.WriteUint32(uint32(totalLength))
	if err != nil {
		return nil, err
	}
	_, err = buf.WriteUint16(uint16(headLength))
	if err != nil {
		return nil, err
	}
	err = buf.WriteByte(byte(msg.Type))
	if err != nil {
		return nil, err
	}
	err = buf.WriteByte(msg.Codec)
	if err != nil {
		return nil, err
	}
	err = buf.WriteByte(msg.Compressor)
	if err != nil {
		return nil, err
	}
	_, err = buf.WriteUint32(uint32(msg.ID))
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(headMapBytes)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(bodyBytes)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeHeapMap(data map[string]string) ([]byte, int) {
	buf := bytes.NewByteBuffer([]byte{})
	for k, v := range data {
		if k == "" {
			_, err := buf.WriteUint16(uint16(0))
			if err != nil {
				return nil, 0
			}
		} else {
			_, err := buf.WriteUint16(uint16(len(k)))
			if err != nil {
				return nil, 0
			}
			_, err = buf.WriteString(k)
			if err != nil {
				return nil, 0
			}
		}

		if v == "" {
			_, err := buf.WriteUint16(uint16(0))
			if err != nil {
				return nil, 0
			}
		} else {
			_, err := buf.WriteUint16(uint16(len(v)))
			if err != nil {
				return nil, 0
			}
			_, err = buf.WriteString(v)
			if err != nil {
				return nil, 0
			}
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
			_, err := in.Read(keyBytes)
			if err != nil {
				return nil
			}
			key = string(keyBytes)
		}

		valueLength := bytes.ReadUInt16(in)
		if valueLength == 0 {
			key = ""
		} else {
			valueBytes := make([]byte, valueLength)
			_, err := in.Read(valueBytes)
			if err != nil {
				return nil
			}
			value = string(valueBytes)
		}

		res[key] = value
		readedLength += 4 + keyLength + valueLength
		fmt.Sprintln("done")
	}
	return res
}
