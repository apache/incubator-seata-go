package codec

import (
	"bytes"
	model2 "github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/transaction"
)

import (
	"vimagination.zapto.org/byteio"
)

// TODO 待重构
func AbstractResultMessageDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := message.AbstractResultMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	msg.ResultCode = message.ResultCode(resultCode)
	totalReadN += 1
	if msg.ResultCode == message.ResultCodeFailed {
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
	result := message.MergedWarpMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	r.ReadInt32()
	totalReadN += 4
	size16, readN, _ = r.ReadInt16()
	totalReadN += readN
	result.Msgs = make([]message.MessageTypeAware, 0)
	for index := 0; index < int(size16); index++ {
		typeCode, _, _ := r.ReadInt16()
		totalReadN += 2
		decoder := getMessageDecoder(message.MessageType(typeCode))
		if decoder != nil {
			msg, readN := decoder(in[totalReadN:])
			totalReadN += readN
			result.Msgs = append(result.Msgs, msg.(message.MessageTypeAware))
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
	result := message.MergeResultMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	r.ReadInt32()
	totalReadN += 4
	size16, readN, _ = r.ReadInt16()
	totalReadN += readN
	result.Msgs = make([]message.MessageTypeAware, 0)

	for index := 0; index < int(size16); index++ {
		typeCode, _, _ := r.ReadInt16()
		totalReadN += 2
		decoder := getMessageDecoder(message.MessageType(typeCode))
		if decoder != nil {
			msg, readN := decoder(in[totalReadN:])
			totalReadN += readN
			result.Msgs = append(result.Msgs, msg.(message.MessageTypeAware))
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
	msg := message.AbstractIdentifyRequest{}

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
	msg := message.AbstractIdentifyResponse{}

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
	msg := message.RegisterRMRequest{}

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
	abstractIdentifyResponse := resp.(message.AbstractIdentifyResponse)
	msg := message.RegisterRMResponse{AbstractIdentifyResponse: abstractIdentifyResponse}
	return msg, totalReadN
}

func RegisterTMRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractIdentifyRequestDecoder(in)
	abstractIdentifyRequest := req.(message.AbstractIdentifyRequest)
	msg := message.RegisterTMRequest{AbstractIdentifyRequest: abstractIdentifyRequest}
	return msg, totalReadN
}

func RegisterTMResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractIdentifyResponseDecoder(in)
	abstractIdentifyResponse := resp.(message.AbstractIdentifyResponse)
	msg := message.RegisterRMResponse{AbstractIdentifyResponse: abstractIdentifyResponse}
	return msg, totalReadN
}

func AbstractTransactionResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := message.AbstractTransactionResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

	return msg, totalReadN
}

func AbstractBranchEndRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := message.AbstractBranchEndRequest{}

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
	msg := message.AbstractBranchEndResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

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
	msg := message.AbstractGlobalEndRequest{}

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
	msg := message.AbstractGlobalEndResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

	globalStatus, _ := r.ReadByte()
	totalReadN += 1
	msg.GlobalStatus = transaction.GlobalStatus(globalStatus)

	return msg, totalReadN
}

func BranchCommitRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractBranchEndRequestDecoder(in)
	abstractBranchEndRequest := req.(message.AbstractBranchEndRequest)
	msg := message.BranchCommitRequest{AbstractBranchEndRequest: abstractBranchEndRequest}
	return msg, totalReadN
}

func BranchCommitResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractBranchEndResponseDecoder(in)
	abstractBranchEndResponse := resp.(message.AbstractBranchEndResponse)
	msg := message.BranchCommitResponse{AbstractBranchEndResponse: abstractBranchEndResponse}
	return msg, totalReadN
}

func BranchRegisterRequestDecoder(in []byte) (interface{}, int) {
	var (
		length32   uint32 = 0
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := message.BranchRegisterRequest{}

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
	msg := message.BranchRegisterResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

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
	msg := message.BranchReportRequest{}

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
	abstractTransactionResponse := resp.(message.AbstractTransactionResponse)
	msg := message.BranchReportResponse{AbstractTransactionResponse: abstractTransactionResponse}
	return msg, totalReadN
}

func BranchRollbackRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractBranchEndRequestDecoder(in)
	abstractBranchEndRequest := req.(message.AbstractBranchEndRequest)
	msg := message.BranchRollbackRequest{AbstractBranchEndRequest: abstractBranchEndRequest}
	return msg, totalReadN
}

func BranchRollbackResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractBranchEndResponseDecoder(in)
	abstractBranchEndResponse := resp.(message.AbstractBranchEndResponse)
	msg := message.BranchRollbackResponse{AbstractBranchEndResponse: abstractBranchEndResponse}
	return msg, totalReadN
}

func GlobalBeginRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := message.GlobalBeginRequest{}

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
	msg := message.GlobalBeginResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

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
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
	msg := message.GlobalCommitRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalCommitResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(message.AbstractGlobalEndResponse)
	msg := message.GlobalCommitResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalLockQueryRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := BranchRegisterRequestDecoder(in)
	branchRegisterRequest := req.(message.BranchRegisterRequest)
	msg := message.GlobalLockQueryRequest{BranchRegisterRequest: branchRegisterRequest}
	return msg, totalReadN
}

func GlobalLockQueryResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := message.GlobalLockQueryResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = message.ResultCode(resultCode)
	if msg.ResultCode == message.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = transaction.TransactionExceptionCode(exceptionCode)

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
	msg := message.GlobalReportRequest{}

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
	msg.GlobalStatus = transaction.GlobalStatus(globalStatus)

	return msg, totalReadN
}

func GlobalReportResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(message.AbstractGlobalEndResponse)
	msg := message.GlobalReportResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalRollbackRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractGlobalEndRequestDecoder(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
	msg := message.GlobalRollbackRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalRollbackResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(message.AbstractGlobalEndResponse)
	msg := message.GlobalRollbackResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalStatusRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractGlobalEndRequestDecoder(in)
	abstractGlobalEndRequest := req.(message.AbstractGlobalEndRequest)
	msg := message.GlobalStatusRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalStatusResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(message.AbstractGlobalEndResponse)
	msg := message.GlobalStatusResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func UndoLogDeleteRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := message.UndoLogDeleteRequest{}

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
	msg.SaveDays = message.MessageType(day)
	totalReadN += readN

	return msg, totalReadN
}
