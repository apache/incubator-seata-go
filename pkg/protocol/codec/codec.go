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

	decoder := getMessageDecoder(typeCode)
	if decoder != nil {
		return decoder(in[2:])
	}
	return nil, 0
}

func getMessageEncoder(typeCode int16) Encoder {
	switch typeCode {
	case protocol.TypeSeataMerge:
		return MergedWarpMessageEncoder
	case protocol.TypeSeataMergeResult:
		return MergeResultMessageEncoder
	case protocol.TypeRegClt:
		return RegisterTMRequestEncoder
	case protocol.TypeRegCltResult:
		return RegisterTMResponseEncoder
	case protocol.TypeRegRm:
		return RegisterRMRequestEncoder
	case protocol.TypeRegRmResult:
		return RegisterRMResponseEncoder
	case protocol.TypeBranchCommit:
		return BranchCommitRequestEncoder
	case protocol.TypeBranchRollback:
		return BranchRollbackRequestEncoder
	case protocol.TypeGlobalReport:
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

func getMergeRequestMessageEncoder(typeCode int16) Encoder {
	switch typeCode {
	case protocol.TypeGlobalBegin:
		return GlobalBeginRequestEncoder
	case protocol.TypeGlobalCommit:
		return GlobalCommitRequestEncoder
	case protocol.TypeGlobalRollback:
		return GlobalRollbackRequestEncoder
	case protocol.TypeGlobalStatus:
		return GlobalStatusRequestEncoder
	case protocol.TypeGlobalLockQuery:
		return GlobalLockQueryRequestEncoder
	case protocol.TypeBranchRegister:
		return BranchRegisterRequestEncoder
	case protocol.TypeBranchStatusReport:
		return BranchReportRequestEncoder
	case protocol.TypeGlobalReport:
		return GlobalReportRequestEncoder
	default:
		break
	}
	return nil
}

func getMergeResponseMessageEncoder(typeCode int16) Encoder {
	switch typeCode {
	case protocol.TypeGlobalBeginResult:
		return GlobalBeginResponseEncoder
	case protocol.TypeGlobalCommitResult:
		return GlobalCommitResponseEncoder
	case protocol.TypeGlobalRollbackResult:
		return GlobalRollbackResponseEncoder
	case protocol.TypeGlobalStatusResult:
		return GlobalStatusResponseEncoder
	case protocol.TypeGlobalLockQueryResult:
		return GlobalLockQueryResponseEncoder
	case protocol.TypeBranchRegisterResult:
		return BranchRegisterResponseEncoder
	case protocol.TypeBranchStatusReportResult:
		return BranchReportResponseEncoder
	case protocol.TypeBranchCommitResult:
		return BranchCommitResponseEncoder
	case protocol.TypeBranchRollbackResult:
		return BranchRollbackResponseEncoder
	case protocol.TypeGlobalReportResult:
		return GlobalReportResponseEncoder
	default:
		break
	}
	return nil
}

func getMessageDecoder(typeCode int16) Decoder {
	switch typeCode {
	case protocol.TypeSeataMerge:
		return MergedWarpMessageDecoder
	case protocol.TypeSeataMergeResult:
		return MergeResultMessageDecoder
	case protocol.TypeRegClt:
		return RegisterTMRequestDecoder
	case protocol.TypeRegCltResult:
		return RegisterTMResponseDecoder
	case protocol.TypeRegRm:
		return RegisterRMRequestDecoder
	case protocol.TypeRegRmResult:
		return RegisterRMResponseDecoder
	case protocol.TypeBranchCommit:
		return BranchCommitRequestDecoder
	case protocol.TypeBranchRollback:
		return BranchRollbackRequestDecoder
	case protocol.TypeGlobalReport:
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

func getMergeRequestMessageDecoder(typeCode int16) Decoder {
	switch typeCode {
	case protocol.TypeGlobalBegin:
		return GlobalBeginRequestDecoder
	case protocol.TypeGlobalCommit:
		return GlobalCommitRequestDecoder
	case protocol.TypeGlobalRollback:
		return GlobalRollbackRequestDecoder
	case protocol.TypeGlobalStatus:
		return GlobalStatusRequestDecoder
	case protocol.TypeGlobalLockQuery:
		return GlobalLockQueryRequestDecoder
	case protocol.TypeBranchRegister:
		return BranchRegisterRequestDecoder
	case protocol.TypeBranchStatusReport:
		return BranchReportRequestDecoder
	case protocol.TypeGlobalReport:
		return GlobalReportRequestDecoder
	default:
		break
	}
	return nil
}

func getMergeResponseMessageDecoder(typeCode int16) Decoder {
	switch typeCode {
	case protocol.TypeGlobalBeginResult:
		return GlobalBeginResponseDecoder
	case protocol.TypeGlobalCommitResult:
		return GlobalCommitResponseDecoder
	case protocol.TypeGlobalRollbackResult:
		return GlobalRollbackResponseDecoder
	case protocol.TypeGlobalStatusResult:
		return GlobalStatusResponseDecoder
	case protocol.TypeGlobalLockQueryResult:
		return GlobalLockQueryResponseDecoder
	case protocol.TypeBranchRegisterResult:
		return BranchRegisterResponseDecoder
	case protocol.TypeBranchStatusReportResult:
		return BranchReportResponseDecoder
	case protocol.TypeBranchCommitResult:
		return BranchCommitResponseDecoder
	case protocol.TypeBranchRollbackResult:
		return BranchRollbackResponseDecoder
	case protocol.TypeGlobalReportResult:
		return GlobalReportResponseDecoder
	default:
		break
	}
	return nil
}
