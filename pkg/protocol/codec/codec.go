package codec

import (
	"bytes"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
)

import (
	"vimagination.zapto.org/byteio"
)

type SerializerType byte

// TODO 待重构
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
	msg := in.(message.MessageTypeAware)
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

	decoder := getMessageDecoder(message.MessageType(typeCode))
	if decoder != nil {
		return decoder(in[2:])
	}
	return nil, 0
}

func getMessageEncoder(typeCode message.MessageType) Encoder {
	switch typeCode {
	case message.MessageType_SeataMerge:
		return MergedWarpMessageEncoder
	case message.MessageType_SeataMergeResult:
		return MergeResultMessageEncoder
	case message.MessageType_RegClt:
		return RegisterTMRequestEncoder
	case message.MessageType_RegCltResult:
		return RegisterTMResponseEncoder
	case message.MessageType_RegRm:
		return RegisterRMRequestEncoder
	case message.MessageType_RegRmResult:
		return RegisterRMResponseEncoder
	case message.MessageType_BranchCommit:
		return BranchCommitRequestEncoder
	case message.MessageType_BranchRollback:
		return BranchRollbackRequestEncoder
	case message.MessageType_GlobalReport:
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

func getMergeRequestMessageEncoder(typeCode message.MessageType) Encoder {
	switch typeCode {
	case message.MessageType_GlobalBegin:
		return GlobalBeginRequestEncoder
	case message.MessageType_GlobalCommit:
		return GlobalCommitRequestEncoder
	case message.MessageType_GlobalRollback:
		return GlobalRollbackRequestEncoder
	case message.MessageType_GlobalStatus:
		return GlobalStatusRequestEncoder
	case message.MessageType_GlobalLockQuery:
		return GlobalLockQueryRequestEncoder
	case message.MessageType_BranchRegister:
		return BranchRegisterRequestEncoder
	case message.MessageType_BranchStatusReport:
		return BranchReportRequestEncoder
	case message.MessageType_GlobalReport:
		return GlobalReportRequestEncoder
	default:
		break
	}
	return nil
}

func getMergeResponseMessageEncoder(typeCode message.MessageType) Encoder {
	switch typeCode {
	case message.MessageType_GlobalBeginResult:
		return GlobalBeginResponseEncoder
	case message.MessageType_GlobalCommitResult:
		return GlobalCommitResponseEncoder
	case message.MessageType_GlobalRollbackResult:
		return GlobalRollbackResponseEncoder
	case message.MessageType_GlobalStatusResult:
		return GlobalStatusResponseEncoder
	case message.MessageType_GlobalLockQueryResult:
		return GlobalLockQueryResponseEncoder
	case message.MessageType_BranchRegisterResult:
		return BranchRegisterResponseEncoder
	case message.MessageType_BranchStatusReportResult:
		return BranchReportResponseEncoder
	case message.MessageType_BranchCommitResult:
		return BranchCommitResponseEncoder
	case message.MessageType_BranchRollbackResult:
		return BranchRollbackResponseEncoder
	case message.MessageType_GlobalReportResult:
		return GlobalReportResponseEncoder
	default:
		break
	}
	return nil
}

func getMessageDecoder(typeCode message.MessageType) Decoder {
	switch typeCode {
	case message.MessageType_SeataMerge:
		return MergedWarpMessageDecoder
	case message.MessageType_SeataMergeResult:
		return MergeResultMessageDecoder
	case message.MessageType_RegClt:
		return RegisterTMRequestDecoder
	case message.MessageType_RegCltResult:
		return RegisterTMResponseDecoder
	case message.MessageType_RegRm:
		return RegisterRMRequestDecoder
	case message.MessageType_RegRmResult:
		return RegisterRMResponseDecoder
	case message.MessageType_BranchCommit:
		return BranchCommitRequestDecoder
	case message.MessageType_BranchRollback:
		return BranchRollbackRequestDecoder
	case message.MessageType_GlobalReport:
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

func getMergeRequestMessageDecoder(typeCode message.MessageType) Decoder {
	switch typeCode {
	case message.MessageType_GlobalBegin:
		return GlobalBeginRequestDecoder
	case message.MessageType_GlobalCommit:
		return GlobalCommitRequestDecoder
	case message.MessageType_GlobalRollback:
		return GlobalRollbackRequestDecoder
	case message.MessageType_GlobalStatus:
		return GlobalStatusRequestDecoder
	case message.MessageType_GlobalLockQuery:
		return GlobalLockQueryRequestDecoder
	case message.MessageType_BranchRegister:
		return BranchRegisterRequestDecoder
	case message.MessageType_BranchStatusReport:
		return BranchReportRequestDecoder
	case message.MessageType_GlobalReport:
		return GlobalReportRequestDecoder
	default:
		break
	}
	return nil
}

func getMergeResponseMessageDecoder(typeCode message.MessageType) Decoder {
	switch typeCode {
	case message.MessageType_GlobalBeginResult:
		return GlobalBeginResponseDecoder
	case message.MessageType_GlobalCommitResult:
		return GlobalCommitResponseDecoder
	case message.MessageType_GlobalRollbackResult:
		return GlobalRollbackResponseDecoder
	case message.MessageType_GlobalStatusResult:
		return GlobalStatusResponseDecoder
	case message.MessageType_GlobalLockQueryResult:
		return GlobalLockQueryResponseDecoder
	case message.MessageType_BranchRegisterResult:
		return BranchRegisterResponseDecoder
	case message.MessageType_BranchStatusReportResult:
		return BranchReportResponseDecoder
	case message.MessageType_BranchCommitResult:
		return BranchCommitResponseDecoder
	case message.MessageType_BranchRollbackResult:
		return BranchRollbackResponseDecoder
	case message.MessageType_GlobalReportResult:
		return GlobalReportResponseDecoder
	default:
		break
	}
	return nil
}
