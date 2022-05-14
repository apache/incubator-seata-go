package codec

import (
	"bytes"
)

import (
	"vimagination.zapto.org/byteio"
)

import (
	model2 "github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/protocol"
)

func AbstractResultMessageDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractResultMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	msg.ResultCode = protocol.ResultCode(resultCode)
	totalReadN += 1
	if msg.ResultCode == protocol.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	return msg, totalReadN
}

func MergedWarpMessageDecoder(in []byte) (interface{}, int) {
	var (
		size16     int16 = 0
		readN            = 0
		totalReadN       = 0
	)
	result := protocol.MergedWarpMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	r.ReadInt32()
	totalReadN += 4
	size16, readN, _ = r.ReadInt16()
	totalReadN += readN
	result.Msgs = make([]protocol.MessageTypeAware, 0)
	for index := 0; index < int(size16); index++ {
		typeCode, _, _ := r.ReadInt16()
		totalReadN += 2
		decoder := getMessageDecoder(protocol.MessageType(typeCode))
		if decoder != nil {
			msg, readN := decoder(in[totalReadN:])
			totalReadN += readN
			result.Msgs = append(result.Msgs, msg.(protocol.MessageTypeAware))
		}
	}
	return result, totalReadN
}

func MergeResultMessageDecoder(in []byte) (interface{}, int) {
	var (
		size16     int16 = 0
		readN            = 0
		totalReadN       = 0
	)
	result := protocol.MergeResultMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	r.ReadInt32()
	totalReadN += 4
	size16, readN, _ = r.ReadInt16()
	totalReadN += readN
	result.Msgs = make([]protocol.MessageTypeAware, 0)

	for index := 0; index < int(size16); index++ {
		typeCode, _, _ := r.ReadInt16()
		totalReadN += 2
		decoder := getMessageDecoder(protocol.MessageType(typeCode))
		if decoder != nil {
			msg, readN := decoder(in[totalReadN:])
			totalReadN += readN
			result.Msgs = append(result.Msgs, msg.(protocol.MessageTypeAware))
		}
	}
	return result, totalReadN
}

func AbstractIdentifyRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractIdentifyRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Version, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ApplicationId, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.TransactionServiceGroup, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ExtraData = make([]byte, int(length16))
		readN, _ := r.Read(msg.ExtraData)
		totalReadN += readN
	}

	return msg, totalReadN
}

func AbstractIdentifyResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractIdentifyResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	identified, _ := r.ReadByte()
	totalReadN += 1
	if identified == byte(1) {
		msg.Identified = true
	} else if identified == byte(0) {
		msg.Identified = false
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Version, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	return msg, totalReadN
}

func RegisterRMRequestDecoder(in []byte) (interface{}, int) {
	var (
		length32   uint32 = 0
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.RegisterRMRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Version, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ApplicationId, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.TransactionServiceGroup, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ExtraData = make([]byte, int(length16))
		readN, _ := r.Read(msg.ExtraData)
		totalReadN += readN
	}

	length32, readN, _ = r.ReadUint32()
	totalReadN += readN
	if length32 > 0 {
		msg.ResourceIds, readN, _ = r.ReadString(int(length32))
		totalReadN += readN
	}

	return msg, totalReadN
}

func RegisterRMResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractIdentifyResponseDecoder(in)
	abstractIdentifyResponse := resp.(protocol.AbstractIdentifyResponse)
	msg := protocol.RegisterRMResponse{AbstractIdentifyResponse: abstractIdentifyResponse}
	return msg, totalReadN
}

func RegisterTMRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractIdentifyRequestDecoder(in)
	abstractIdentifyRequest := req.(protocol.AbstractIdentifyRequest)
	msg := protocol.RegisterTMRequest{AbstractIdentifyRequest: abstractIdentifyRequest}
	return msg, totalReadN
}

func RegisterTMResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractIdentifyResponseDecoder(in)
	abstractIdentifyResponse := resp.(protocol.AbstractIdentifyResponse)
	msg := protocol.RegisterRMResponse{AbstractIdentifyResponse: abstractIdentifyResponse}
	return msg, totalReadN
}

func AbstractTransactionResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractTransactionResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocol.ResultCode(resultCode)
	if msg.ResultCode == protocol.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = model2.TransactionExceptionCode(exceptionCode)

	return msg, totalReadN
}

func AbstractBranchEndRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractBranchEndRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Xid, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	msg.BranchId, _, _ = r.ReadInt64()
	totalReadN += 8
	branchType, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchType = model2.BranchType(branchType)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceId, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ApplicationData = make([]byte, int(length16))
		readN, _ := r.Read(msg.ApplicationData)
		totalReadN += readN
	}

	return msg, totalReadN
}

func AbstractBranchEndResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractBranchEndResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocol.ResultCode(resultCode)
	if msg.ResultCode == protocol.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = model2.TransactionExceptionCode(exceptionCode)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Xid, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	msg.BranchId, _, _ = r.ReadInt64()
	totalReadN += 8
	branchStatus, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchStatus = model2.BranchStatus(branchStatus)

	return msg, totalReadN
}

func AbstractGlobalEndRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractGlobalEndRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Xid, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ExtraData = make([]byte, int(length16))
		readN, _ := r.Read(msg.ExtraData)
		totalReadN += readN
	}

	return msg, totalReadN
}

func AbstractGlobalEndResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.AbstractGlobalEndResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocol.ResultCode(resultCode)
	if msg.ResultCode == protocol.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = model2.TransactionExceptionCode(exceptionCode)

	globalStatus, _ := r.ReadByte()
	totalReadN += 1
	msg.GlobalStatus = model2.GlobalStatus(globalStatus)

	return msg, totalReadN
}

func BranchCommitRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractBranchEndRequestDecoder(in)
	abstractBranchEndRequest := req.(protocol.AbstractBranchEndRequest)
	msg := protocol.BranchCommitRequest{AbstractBranchEndRequest: abstractBranchEndRequest}
	return msg, totalReadN
}

func BranchCommitResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractBranchEndResponseDecoder(in)
	abstractBranchEndResponse := resp.(protocol.AbstractBranchEndResponse)
	msg := protocol.BranchCommitResponse{AbstractBranchEndResponse: abstractBranchEndResponse}
	return msg, totalReadN
}

func BranchRegisterRequestDecoder(in []byte) (interface{}, int) {
	var (
		length32   uint32 = 0
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.BranchRegisterRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Xid, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	branchType, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchType = model2.BranchType(branchType)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceId, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length32, readN, _ = r.ReadUint32()
	totalReadN += readN
	if length32 > 0 {
		msg.LockKey, readN, _ = r.ReadString(int(length32))
		totalReadN += readN
	}

	length32, readN, _ = r.ReadUint32()
	totalReadN += readN
	if length32 > 0 {
		msg.ApplicationData = make([]byte, int(length32))
		readN, _ := r.Read(msg.ApplicationData)
		totalReadN += readN
	}

	return msg, totalReadN
}

func BranchRegisterResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.BranchRegisterResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocol.ResultCode(resultCode)
	if msg.ResultCode == protocol.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = model2.TransactionExceptionCode(exceptionCode)

	msg.BranchId, readN, _ = r.ReadInt64()
	totalReadN += readN

	return msg, totalReadN
}

func BranchReportRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.BranchReportRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Xid, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	msg.BranchId, _, _ = r.ReadInt64()
	branchStatus, _ := r.ReadByte()
	msg.Status = model2.BranchStatus(branchStatus)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceId, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ApplicationData = make([]byte, int(length16))
		readN, _ := r.Read(msg.ApplicationData)
		totalReadN += readN
	}

	branchType, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchType = model2.BranchType(branchType)

	return msg, totalReadN
}

func BranchReportResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractTransactionResponseDecoder(in)
	abstractTransactionResponse := resp.(protocol.AbstractTransactionResponse)
	msg := protocol.BranchReportResponse{AbstractTransactionResponse: abstractTransactionResponse}
	return msg, totalReadN
}

func BranchRollbackRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractBranchEndRequestDecoder(in)
	abstractBranchEndRequest := req.(protocol.AbstractBranchEndRequest)
	msg := protocol.BranchRollbackRequest{AbstractBranchEndRequest: abstractBranchEndRequest}
	return msg, totalReadN
}

func BranchRollbackResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractBranchEndResponseDecoder(in)
	abstractBranchEndResponse := resp.(protocol.AbstractBranchEndResponse)
	msg := protocol.BranchRollbackResponse{AbstractBranchEndResponse: abstractBranchEndResponse}
	return msg, totalReadN
}

func GlobalBeginRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.GlobalBeginRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	timeout, readN, _ := r.ReadInt32()
	totalReadN += readN
	msg.Timeout = timeout

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.TransactionName, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	return msg, totalReadN
}

func GlobalBeginResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.GlobalBeginResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocol.ResultCode(resultCode)
	if msg.ResultCode == protocol.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = model2.TransactionExceptionCode(exceptionCode)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Xid, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ExtraData = make([]byte, int(length16))
		readN, _ := r.Read(msg.ExtraData)
		totalReadN += readN
	}

	return msg, totalReadN
}

func GlobalCommitRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractGlobalEndRequestDecoder(in)
	abstractGlobalEndRequest := req.(protocol.AbstractGlobalEndRequest)
	msg := protocol.GlobalCommitRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalCommitResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocol.AbstractGlobalEndResponse)
	msg := protocol.GlobalCommitResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalLockQueryRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := BranchRegisterRequestDecoder(in)
	branchRegisterRequest := req.(protocol.BranchRegisterRequest)
	msg := protocol.GlobalLockQueryRequest{BranchRegisterRequest: branchRegisterRequest}
	return msg, totalReadN
}

func GlobalLockQueryResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.GlobalLockQueryResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocol.ResultCode(resultCode)
	if msg.ResultCode == protocol.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = model2.TransactionExceptionCode(exceptionCode)

	lockable, readN, _ := r.ReadUint16()
	totalReadN += readN
	if lockable == uint16(1) {
		msg.Lockable = true
	} else if lockable == uint16(0) {
		msg.Lockable = false
	}

	return msg, totalReadN
}

func GlobalReportRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.GlobalReportRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.Xid, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ExtraData = make([]byte, int(length16))
		readN, _ := r.Read(msg.ExtraData)
		totalReadN += readN
	}

	globalStatus, _ := r.ReadByte()
	totalReadN += 1
	msg.GlobalStatus = model2.GlobalStatus(globalStatus)

	return msg, totalReadN
}

func GlobalReportResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocol.AbstractGlobalEndResponse)
	msg := protocol.GlobalReportResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalRollbackRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractGlobalEndRequestDecoder(in)
	abstractGlobalEndRequest := req.(protocol.AbstractGlobalEndRequest)
	msg := protocol.GlobalRollbackRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalRollbackResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocol.AbstractGlobalEndResponse)
	msg := protocol.GlobalRollbackResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalStatusRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractGlobalEndRequestDecoder(in)
	abstractGlobalEndRequest := req.(protocol.AbstractGlobalEndRequest)
	msg := protocol.GlobalStatusRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalStatusResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocol.AbstractGlobalEndResponse)
	msg := protocol.GlobalStatusResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func UndoLogDeleteRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocol.UndoLogDeleteRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	branchType, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchType = model2.BranchType(branchType)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceId, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	day, readN, _ := r.ReadInt16()
	msg.SaveDays = protocol.MessageType(day)
	totalReadN += readN

	return msg, totalReadN
}
