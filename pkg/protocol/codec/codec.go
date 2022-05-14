package codec

import (
	"bytes"
)

import (
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/utils/log"
)

type SerializerType byte

const (
	SEATA    = byte(0x1)
	PROTOBUF = byte(0x2)
	KRYO     = byte(0x4)
	FST      = byte(0x8)
)

type Encoder func(in interface{}) []byte

type Decoder func(in []byte) (interface{}, int)

func MessageEncoder(codecType byte, in interface{}) []byte {
	switch codecType {
	case SEATA:
		return SeataEncoder(in)
	default:
		log.Errorf("not support codecType, %s", codecType)
		return nil
	}
}

func MessageDecoder(codecType byte, in []byte) (interface{}, int) {
	switch codecType {
	case SEATA:
		return SeataDecoder(in)
	default:
		log.Errorf("not support codecType, %s", codecType)
		return nil, 0
	}
}

func SeataEncoder(in interface{}) []byte {
	var result = make([]byte, 0)
	msg := in.(protocol.MessageTypeAware)
	typeCode := msg.GetTypeCode()
	encoder := getMessageEncoder(typeCode)

	typeC := uint16(typeCode)
	if encoder != nil {
		body := encoder(in)
		result = append(result, []byte{byte(typeC >> 8), byte(typeC)}...)
		result = append(result, body...)
	}
	return result
}

func SeataDecoder(in []byte) (interface{}, int) {
	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	typeCode, _, _ := r.ReadInt16()

	decoder := getMessageDecoder(protocol.MessageType(typeCode))
	if decoder != nil {
		return decoder(in[2:])
	}
	return nil, 0
}

func getMessageEncoder(typeCode protocol.MessageType) Encoder {
	switch typeCode {
	case protocol.MessageTypeSeataMerge:
		return MergedWarpMessageEncoder
	case protocol.MessageTypeSeataMergeResult:
		return MergeResultMessageEncoder
	case protocol.MessageTypeRegClt:
		return RegisterTMRequestEncoder
	case protocol.MessageTypeRegCltResult:
		return RegisterTMResponseEncoder
	case protocol.MessageTypeRegRm:
		return RegisterRMRequestEncoder
	case protocol.MessageTypeRegRmResult:
		return RegisterRMResponseEncoder
	case protocol.MessageTypeBranchCommit:
		return BranchCommitRequestEncoder
	case protocol.MessageTypeBranchRollback:
		return BranchRollbackRequestEncoder
	case protocol.MessageTypeGlobalReport:
		return GlobalReportRequestEncoder
	default:
		var encoder Encoder
		encoder = getMergeRequestMessageEncoder(typeCode)
		if encoder != nil {
			return encoder
		}
		encoder = getMergeResponseMessageEncoder(typeCode)
		if encoder != nil {
			return encoder
		}
		log.Errorf("not support typeCode, %d", typeCode)
		return nil
	}
}

func getMergeRequestMessageEncoder(typeCode protocol.MessageType) Encoder {
	switch typeCode {
	case protocol.MessageTypeGlobalBegin:
		return GlobalBeginRequestEncoder
	case protocol.MessageTypeGlobalCommit:
		return GlobalCommitRequestEncoder
	case protocol.MessageTypeGlobalRollback:
		return GlobalRollbackRequestEncoder
	case protocol.MessageTypeGlobalStatus:
		return GlobalStatusRequestEncoder
	case protocol.MessageTypeGlobalLockQuery:
		return GlobalLockQueryRequestEncoder
	case protocol.MessageTypeBranchRegister:
		return BranchRegisterRequestEncoder
	case protocol.MessageTypeBranchStatusReport:
		return BranchReportRequestEncoder
	case protocol.MessageTypeGlobalReport:
		return GlobalReportRequestEncoder
	default:
		break
	}
	return nil
}

func getMergeResponseMessageEncoder(typeCode protocol.MessageType) Encoder {
	switch typeCode {
	case protocol.MessageTypeGlobalBeginResult:
		return GlobalBeginResponseEncoder
	case protocol.MessageTypeGlobalCommitResult:
		return GlobalCommitResponseEncoder
	case protocol.MessageTypeGlobalRollbackResult:
		return GlobalRollbackResponseEncoder
	case protocol.MessageTypeGlobalStatusResult:
		return GlobalStatusResponseEncoder
	case protocol.MessageTypeGlobalLockQueryResult:
		return GlobalLockQueryResponseEncoder
	case protocol.MessageTypeBranchRegisterResult:
		return BranchRegisterResponseEncoder
	case protocol.MessageTypeBranchStatusReportResult:
		return BranchReportResponseEncoder
	case protocol.MessageTypeBranchCommitResult:
		return BranchCommitResponseEncoder
	case protocol.MessageTypeBranchRollbackResult:
		return BranchRollbackResponseEncoder
	case protocol.MessageTypeGlobalReportResult:
		return GlobalReportResponseEncoder
	default:
		break
	}
	return nil
}

func getMessageDecoder(typeCode protocol.MessageType) Decoder {
	switch typeCode {
	case protocol.MessageTypeSeataMerge:
		return MergedWarpMessageDecoder
	case protocol.MessageTypeSeataMergeResult:
		return MergeResultMessageDecoder
	case protocol.MessageTypeRegClt:
		return RegisterTMRequestDecoder
	case protocol.MessageTypeRegCltResult:
		return RegisterTMResponseDecoder
	case protocol.MessageTypeRegRm:
		return RegisterRMRequestDecoder
	case protocol.MessageTypeRegRmResult:
		return RegisterRMResponseDecoder
	case protocol.MessageTypeBranchCommit:
		return BranchCommitRequestDecoder
	case protocol.MessageTypeBranchRollback:
		return BranchRollbackRequestDecoder
	case protocol.MessageTypeGlobalReport:
		return GlobalReportRequestDecoder
	default:
		var Decoder Decoder
		Decoder = getMergeRequestMessageDecoder(typeCode)
		if Decoder != nil {
			return Decoder
		}
		Decoder = getMergeResponseMessageDecoder(typeCode)
		if Decoder != nil {
			return Decoder
		}
		log.Errorf("not support typeCode, %d", typeCode)
		return nil
	}
}

func getMergeRequestMessageDecoder(typeCode protocol.MessageType) Decoder {
	switch typeCode {
	case protocol.MessageTypeGlobalBegin:
		return GlobalBeginRequestDecoder
	case protocol.MessageTypeGlobalCommit:
		return GlobalCommitRequestDecoder
	case protocol.MessageTypeGlobalRollback:
		return GlobalRollbackRequestDecoder
	case protocol.MessageTypeGlobalStatus:
		return GlobalStatusRequestDecoder
	case protocol.MessageTypeGlobalLockQuery:
		return GlobalLockQueryRequestDecoder
	case protocol.MessageTypeBranchRegister:
		return BranchRegisterRequestDecoder
	case protocol.MessageTypeBranchStatusReport:
		return BranchReportRequestDecoder
	case protocol.MessageTypeGlobalReport:
		return GlobalReportRequestDecoder
	default:
		break
	}
	return nil
}

func getMergeResponseMessageDecoder(typeCode protocol.MessageType) Decoder {
	switch typeCode {
	case protocol.MessageTypeGlobalBeginResult:
		return GlobalBeginResponseDecoder
	case protocol.MessageTypeGlobalCommitResult:
		return GlobalCommitResponseDecoder
	case protocol.MessageTypeGlobalRollbackResult:
		return GlobalRollbackResponseDecoder
	case protocol.MessageTypeGlobalStatusResult:
		return GlobalStatusResponseDecoder
	case protocol.MessageTypeGlobalLockQueryResult:
		return GlobalLockQueryResponseDecoder
	case protocol.MessageTypeBranchRegisterResult:
		return BranchRegisterResponseDecoder
	case protocol.MessageTypeBranchStatusReportResult:
		return BranchReportResponseDecoder
	case protocol.MessageTypeBranchCommitResult:
		return BranchCommitResponseDecoder
	case protocol.MessageTypeBranchRollbackResult:
		return BranchRollbackResponseDecoder
	case protocol.MessageTypeGlobalReportResult:
		return GlobalReportResponseDecoder
	default:
		break
	}
	return nil
}
