package readwriter

import (
	"bytes"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/pkg/errors"
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal/codec"
)

/**
 * <pre>
 * 0     1     2     3     4     5     6     7     8     9    10     11    12    13    14    15    16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |   magic   |Proto|     Full length       |    Head   | Msg |Seria|Compr|     RequestID         |
 * |   code    |colVer|    (head+body)       |   Length  |Type |lizer|ess  |                       |
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

var (
	RpcPkgHandler = &RpcPackageHandler{}
)

type RpcPackageHandler struct{}

func (p *RpcPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	r := byteio.BigEndianReader{Reader: bytes.NewReader(data)}

	b0, _ := r.ReadByte()
	b1, _ := r.ReadByte()

	if b0 != protocal.MAGIC_CODE_BYTES[0] || b1 != protocal.MAGIC_CODE_BYTES[1] {
		return nil, 0, errors.Errorf("Unknown magic code: %b,%b", b0, b1)
	}

	r.ReadByte()
	// TODO  check version compatible here

	fullLength, _, _ := r.ReadInt32()
	headLength, _, _ := r.ReadInt16()
	messageType, _ := r.ReadByte()
	codecType, _ := r.ReadByte()
	compressorType, _ := r.ReadByte()
	requestID, _, _ := r.ReadInt32()

	rpcMessage := protocal.RpcMessage{
		Codec:       codecType,
		ID:          requestID,
		Compressor:  compressorType,
		MessageType: messageType,
	}

	headMapLength := headLength - protocal.V1_HEAD_LENGTH
	if headMapLength > 0 {
		rpcMessage.HeadMap = headMapDecode(data[protocal.V1_HEAD_LENGTH+1 : headMapLength])
	}

	if messageType == protocal.MSGTYPE_HEARTBEAT_REQUEST {
		rpcMessage.Body = protocal.HeartBeatMessagePing
	} else if messageType == protocal.MSGTYPE_HEARTBEAT_RESPONSE {
		rpcMessage.Body = protocal.HeartBeatMessagePong
	} else {
		bodyLength := fullLength - int32(headLength)
		if bodyLength > 0 {
			//todo compress

			msg, _ := codec.MessageDecoder(codecType, data[headLength:])
			rpcMessage.Body = msg
		}
	}

	return rpcMessage, int(fullLength), nil
}

func (p *RpcPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	var result = make([]byte, 0)
	msg := pkg.(protocal.RpcMessage)

	fullLength := protocal.V1_HEAD_LENGTH
	headLength := protocal.V1_HEAD_LENGTH

	var b bytes.Buffer
	w := byteio.BigEndianWriter{Writer: &b}

	result = append(result, protocal.MAGIC_CODE_BYTES[:2]...)
	result = append(result, protocal.VERSION)

	w.WriteByte(msg.MessageType)
	w.WriteByte(msg.Codec)
	w.WriteByte(msg.Compressor)
	w.WriteInt32(msg.ID)

	if msg.HeadMap != nil && len(msg.HeadMap) > 0 {
		headMapBytes, headMapLength := headMapEncode(msg.HeadMap)
		headLength += headMapLength
		fullLength += headMapLength
		w.Write(headMapBytes)
	}

	if msg.MessageType != protocal.MSGTYPE_HEARTBEAT_REQUEST &&
		msg.MessageType != protocal.MSGTYPE_HEARTBEAT_RESPONSE {
		bodyBytes := codec.MessageEncoder(msg.Codec, msg.Body)
		fullLength += len(bodyBytes)
		w.Write(bodyBytes)
	}

	fullLen := int32(fullLength)
	headLen := int16(headLength)
	result = append(result, []byte{byte(fullLen >> 24), byte(fullLen >> 16), byte(fullLen >> 8), byte(fullLen)}...)
	result = append(result, []byte{byte(headLen >> 8), byte(headLen)}...)
	result = append(result, b.Bytes()...)

	return result, nil
}

func headMapDecode(data []byte) map[string]string {
	mp := make(map[string]string)
	size := len(data)
	if size == 0 {
		return mp
	}
	r := byteio.BigEndianReader{Reader: bytes.NewReader(data)}

	readLength := 0
	for readLength < size {
		var key, value string
		lengthK, _, _ := r.ReadUint16()
		if lengthK < 0 {
			break
		} else if lengthK == 0 {
			key = ""
		} else {
			key, _, _ = r.ReadString(int(lengthK))
		}

		lengthV, _, _ := r.ReadUint16()
		if lengthV < 0 {
			break
		} else if lengthV == 0 {
			value = ""
		} else {
			value, _, _ = r.ReadString(int(lengthV))
		}

		mp[key] = value
		readLength += int(lengthK + lengthV)
	}
	return mp
}

func headMapEncode(data map[string]string) ([]byte, int) {
	var b bytes.Buffer

	w := byteio.BigEndianWriter{Writer: &b}
	for k, v := range data {
		if k == "" {
			w.WriteUint16(0)
		} else {
			w.WriteUint16(uint16(len(k)))
			w.WriteString(k)
		}

		if v == "" {
			w.WriteUint16(0)
		} else {
			w.WriteUint16(uint16(len(v)))
			w.WriteString(v)
		}
	}
	return b.Bytes(), b.Len()
}
