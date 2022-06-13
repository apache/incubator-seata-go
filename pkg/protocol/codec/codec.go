package codec

import (
	"bytes"
	"sync"
)

import (
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
)

type CodecType byte

// TODO 待重构
const (
	CodeTypeSeata    = CodecType(0x1)
	CodeTypeProtobuf = CodecType(0x2)
	CodeTypeKRYO     = CodecType(0x4)
	CodeTypeFST      = CodecType(0x8)
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
	var result = make([]byte, 0)
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
