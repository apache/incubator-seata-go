package getty

import (
	"bytes"
	"encoding/binary"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/pkg/errors"

	"vimagination.zapto.org/byteio"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/protocol/codec"
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
	SeataV1PackageHeaderReservedLength = 16
)

var (
	// RpcPkgHandler
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
	MessageType  protocol.MessageType
	CodecType    byte
	CompressType byte
	ID           uint32
	Meta         map[string]string
	BodyLength   uint32
}

func (h *SeataV1PackageHeader) Unmarshal(buf *bytes.Buffer) (int, error) {
	bufLen := buf.Len()
	if bufLen < SeataV1PackageHeaderReservedLength {
		return 0, ErrNotEnoughStream
	}

	// magic
	if err := binary.Read(buf, binary.BigEndian, &(h.Magic0)); err != nil {
		return 0, err
	}
	if err := binary.Read(buf, binary.BigEndian, &(h.Magic1)); err != nil {
		return 0, err
	}
	if h.Magic0 != protocol.MAGIC_CODE_BYTES[0] || h.Magic1 != protocol.MAGIC_CODE_BYTES[1] {
		return 0, ErrIllegalMagic
	}
	// version
	if err := binary.Read(buf, binary.BigEndian, &(h.Version)); err != nil {
		return 0, err
	}
	// TODO  check version compatible here

	// total length
	if err := binary.Read(buf, binary.BigEndian, &(h.TotalLength)); err != nil {
		return 0, err
	}
	// head length
	if err := binary.Read(buf, binary.BigEndian, &(h.HeadLength)); err != nil {
		return 0, err
	}
	// message type
	if err := binary.Read(buf, binary.BigEndian, &(h.MessageType)); err != nil {
		return 0, err
	}
	// codec type
	if err := binary.Read(buf, binary.BigEndian, &(h.CodecType)); err != nil {
		return 0, err
	}
	// compress type
	if err := binary.Read(buf, binary.BigEndian, &(h.CompressType)); err != nil {
		return 0, err
	}
	// id
	if err := binary.Read(buf, binary.BigEndian, &(h.ID)); err != nil {
		return 0, err
	}
	// todo meta map
	if h.HeadLength > SeataV1PackageHeaderReservedLength {
		headMapLength := h.HeadLength - SeataV1PackageHeaderReservedLength
		h.Meta = headMapDecode(buf.Bytes()[:headMapLength])
	}
	h.BodyLength = h.TotalLength - uint32(h.HeadLength)

	return int(h.TotalLength), nil
}

// Read read binary data from to rpc message
func (p *RpcPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	var header SeataV1PackageHeader

	buf := bytes.NewBuffer(data)
	_, err := header.Unmarshal(buf)
	if err != nil {
		if err == ErrNotEnoughStream {
			// getty case2
			return nil, 0, nil
		}
		// getty case1
		return nil, 0, err
	}
	if uint32(len(data)) < header.TotalLength {
		// get case3
		return nil, int(header.TotalLength), nil
	}

	//r := byteio.BigEndianReader{Reader: bytes.NewReader(data)}
	rpcMessage := protocol.RpcMessage{
		Codec:       header.CodecType,
		ID:          int32(header.ID),
		Compressor:  header.CompressType,
		MessageType: header.MessageType,
		HeadMap:     header.Meta,
	}

	if header.MessageType == protocol.MSGTypeHeartbeatRequest {
		rpcMessage.Body = protocol.HeartBeatMessagePing
	} else if header.MessageType == protocol.MSGTypeHeartbeatResponse {
		rpcMessage.Body = protocol.HeartBeatMessagePong
	} else {
		if header.BodyLength > 0 {
			//todo compress
			msg, _ := codec.MessageDecoder(header.CodecType, data[header.HeadLength:])
			rpcMessage.Body = msg
		}
	}

	return rpcMessage, int(header.TotalLength), nil
}

// Write write rpc message to binary data
func (p *RpcPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	msg, ok := pkg.(protocol.RpcMessage)
	if !ok {
		return nil, ErrInvalidPackage
	}

	fullLength := protocol.V1HeadLength
	headLength := protocol.V1HeadLength
	var result = make([]byte, 0, fullLength)

	var b bytes.Buffer
	w := byteio.BigEndianWriter{Writer: &b}

	result = append(result, protocol.MAGIC_CODE_BYTES[:2]...)
	result = append(result, protocol.VERSION)

	w.WriteByte(byte(msg.MessageType))
	w.WriteByte(msg.Codec)
	w.WriteByte(msg.Compressor)
	w.WriteInt32(msg.ID)

	if msg.HeadMap != nil && len(msg.HeadMap) > 0 {
		headMapBytes, headMapLength := headMapEncode(msg.HeadMap)
		headLength += headMapLength
		fullLength += headMapLength
		w.Write(headMapBytes)
	}

	if msg.MessageType != protocol.MSGTypeHeartbeatRequest &&
		msg.MessageType != protocol.MSGTypeHeartbeatResponse {

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
	size := len(data)
	if size == 0 {
		return nil
	}

	mp := make(map[string]string)
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
