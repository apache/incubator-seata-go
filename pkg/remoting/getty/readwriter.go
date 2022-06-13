package getty

import (
	"fmt"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/fagongzi/goetty"

	"github.com/pkg/errors"
)

import (
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
	in := goetty.NewByteBuf(len(data))
	in.Write(data)

	header := SeataV1PackageHeader{}
	if in.Readable() < Seatav1HeaderLength {
		return nil, 0, fmt.Errorf("invalid package length")
	}

	magic0 := codec.ReadByte(in)
	magic1 := codec.ReadByte(in)
	if magic0 != magics[0] || magic1 != magics[1] {
		return nil, 0, fmt.Errorf("codec decode not found magic offset")
	}

	header.Magic0 = magic0
	header.Magic1 = magic1
	header.Version = codec.ReadByte(in)
	// length of head and body
	header.TotalLength = codec.ReadUInt32(in)
	header.HeadLength = codec.ReadUInt16(in)
	header.MessageType = message.GettyRequestType(codec.ReadByte(in))
	header.CodecType = codec.ReadByte(in)
	header.CompressType = codec.ReadByte(in)
	header.RequestID = codec.ReadUInt32(in)

	headMapLength := header.HeadLength - Seatav1HeaderLength
	header.Meta = decodeHeapMap(in, headMapLength)
	header.BodyLength = header.TotalLength - uint32(header.HeadLength)

	if uint32(len(data)) < header.TotalLength {
		return nil, int(header.TotalLength), nil
	}

	//r := byteio.BigEndianReader{Reader: bytes.NewReader(data)}
	rpcMessage := message.RpcMessage{
		Codec:      header.CodecType,
		ID:         int32(header.RequestID),
		Compressor: header.CompressType,
		Type:       header.MessageType,
		HeadMap:    header.Meta,
	}

	if header.MessageType == message.GettyRequestType_HeartbeatRequest {
		rpcMessage.Body = message.HeartBeatMessagePing
	} else if header.MessageType == message.GettyRequestType_HeartbeatResponse {
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
	if msg.Type != message.GettyRequestType_HeartbeatRequest &&
		msg.Type != message.GettyRequestType_HeartbeatResponse {
		bodyBytes = codec.GetCodecManager().Encode(codec.CodecType(msg.Codec), msg.Body)
		totalLength += len(bodyBytes)
	}

	buf := goetty.NewByteBuf(0)
	buf.WriteByte(message.MAGIC_CODE_BYTES[0])
	buf.WriteByte(message.MAGIC_CODE_BYTES[1])
	buf.WriteByte(message.VERSION)
	buf.WriteUInt32(uint32(totalLength))
	buf.WriteUInt16(uint16(headLength))
	buf.WriteByte(byte(msg.Type))
	buf.WriteByte(msg.Codec)
	buf.WriteByte(msg.Compressor)
	buf.WriteUInt32(uint32(msg.ID))
	buf.Write(headMapBytes)
	buf.Write(bodyBytes)

	return buf.RawBuf(), nil
}

func encodeHeapMap(data map[string]string) ([]byte, int) {
	buf := goetty.NewByteBuf(0)
	for k, v := range data {
		if k == "" {
			buf.WriteUInt16(uint16(0))
		} else {
			buf.WriteUInt16(uint16(len(k)))
			buf.WriteString(k)
		}

		if v == "" {
			buf.WriteUInt16(uint16(0))
		} else {
			buf.WriteUInt16(uint16(len(v)))
			buf.WriteString(v)
		}
	}
	res := buf.RawBuf()
	return res, len(res)
}

func decodeHeapMap(in *goetty.ByteBuf, length uint16) map[string]string {
	res := make(map[string]string, 0)
	if length == 0 {
		return res
	}

	readedLength := uint16(0)
	for readedLength < length {
		var key, value string
		keyLength := codec.ReadUInt16(in)
		if keyLength == 0 {
			key = ""
		} else {
			keyBytes := make([]byte, keyLength)
			keyBytes = codec.Read(in, keyBytes)
			key = string(keyBytes)
		}

		valueLength := codec.ReadUInt16(in)
		if valueLength == 0 {
			key = ""
		} else {
			valueBytes := make([]byte, valueLength)
			valueBytes = codec.Read(in, valueBytes)
			value = string(valueBytes)
		}

		res[key] = value
		readedLength += 4 + keyLength + valueLength
		fmt.Sprintln("done")
	}
	return res
}
