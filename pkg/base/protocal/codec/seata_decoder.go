package codec

import (
	"bytes"
)

import (
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
)

func AbstractResultMessageDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.AbstractResultMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	msg.ResultCode = protocal.ResultCode(resultCode)
	totalReadN += 1
	if msg.ResultCode == protocal.ResultCodeFailed {
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
	result := protocal.MergedWarpMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	r.ReadInt32()
	totalReadN += 4
	size16, readN, _ = r.ReadInt16()
	totalReadN += readN
	result.Msgs = make([]protocal.MessageTypeAware, 0)
	for index := 0; index < int(size16); index++ {
		typeCode16 := in[totalReadN : totalReadN+2]
		typeCode := int16(uint16(typeCode16[1]) | uint16(typeCode16[0])<<8)
		totalReadN += 2
		decoder := getMessageDecoder(typeCode)
		if decoder != nil {
			msg, readN := decoder(in[totalReadN:])
			totalReadN += readN
			result.Msgs = append(result.Msgs, msg.(protocal.MessageTypeAware))
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
	result := protocal.MergeResultMessage{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	r.ReadInt32()
	totalReadN += 4
	size16, readN, _ = r.ReadInt16()
	totalReadN += readN
	result.Msgs = make([]protocal.MessageTypeAware, 0)

	for index := 0; index < int(size16); index++ {
		typeCode16 := in[totalReadN : totalReadN+2]
		typeCode := int16(uint16(typeCode16[1]) | uint16(typeCode16[0])<<8)
		totalReadN += 2
		decoder := getMessageDecoder(typeCode)
		if decoder != nil {
			msg, readN := decoder(in[totalReadN:])
			totalReadN += readN
			result.Msgs = append(result.Msgs, msg.(protocal.MessageTypeAware))
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
	msg := protocal.AbstractIdentifyRequest{}

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
		msg.ApplicationID, readN, _ = r.ReadString(int(length16))
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
	msg := protocal.AbstractIdentifyResponse{}

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
	msg := protocal.RegisterRMRequest{}

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
		msg.ApplicationID, readN, _ = r.ReadString(int(length16))
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
		msg.ResourceIDs, readN, _ = r.ReadString(int(length32))
		totalReadN += readN
	}

	return msg, totalReadN
}

func RegisterRMResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractIdentifyResponseDecoder(in)
	abstractIdentifyResponse := resp.(protocal.AbstractIdentifyResponse)
	msg := protocal.RegisterRMResponse{AbstractIdentifyResponse: abstractIdentifyResponse}
	return msg, totalReadN
}

func RegisterTMRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractIdentifyRequestDecoder(in)
	abstractIdentifyRequest := req.(protocal.AbstractIdentifyRequest)
	msg := protocal.RegisterTMRequest{AbstractIdentifyRequest: abstractIdentifyRequest}
	return msg, totalReadN
}

func RegisterTMResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractIdentifyResponseDecoder(in)
	abstractIdentifyResponse := resp.(protocal.AbstractIdentifyResponse)
	msg := protocal.RegisterRMResponse{AbstractIdentifyResponse: abstractIdentifyResponse}
	return msg, totalReadN
}

func AbstractTransactionResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.AbstractTransactionResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocal.ResultCode(resultCode)
	if msg.ResultCode == protocal.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = meta.TransactionExceptionCode(exceptionCode)

	return msg, totalReadN
}

func AbstractBranchEndRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		length32   uint32 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.AbstractBranchEndRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.XID, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	msg.BranchID, _, _ = r.ReadInt64()
	totalReadN += 8
	branchType, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchType = meta.BranchType(branchType)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceID, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	length32, readN, _ = r.ReadUint32()
	totalReadN += readN
	if length16 > 0 {
		msg.ApplicationData = make([]byte, int(length32))
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
	msg := protocal.AbstractBranchEndResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocal.ResultCode(resultCode)
	if msg.ResultCode == protocal.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = meta.TransactionExceptionCode(exceptionCode)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.XID, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	msg.BranchID, _, _ = r.ReadInt64()
	totalReadN += 8
	branchStatus, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchStatus = meta.BranchStatus(branchStatus)

	return msg, totalReadN
}

func AbstractGlobalEndRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.AbstractGlobalEndRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.XID, readN, _ = r.ReadString(int(length16))
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
	msg := protocal.AbstractGlobalEndResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocal.ResultCode(resultCode)
	if msg.ResultCode == protocal.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = meta.TransactionExceptionCode(exceptionCode)

	globalStatus, _ := r.ReadByte()
	totalReadN += 1
	msg.GlobalStatus = meta.GlobalStatus(globalStatus)

	return msg, totalReadN
}

func BranchCommitRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractBranchEndRequestDecoder(in)
	abstractBranchEndRequest := req.(protocal.AbstractBranchEndRequest)
	msg := protocal.BranchCommitRequest{AbstractBranchEndRequest: abstractBranchEndRequest}
	return msg, totalReadN
}

func BranchCommitResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractBranchEndResponseDecoder(in)
	abstractBranchEndResponse := resp.(protocal.AbstractBranchEndResponse)
	msg := protocal.BranchCommitResponse{AbstractBranchEndResponse: abstractBranchEndResponse}
	return msg, totalReadN
}

func BranchRegisterRequestDecoder(in []byte) (interface{}, int) {
	var (
		length32   uint32 = 0
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.BranchRegisterRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.XID, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	branchType, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchType = meta.BranchType(branchType)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceID, readN, _ = r.ReadString(int(length16))
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
	msg := protocal.BranchRegisterResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocal.ResultCode(resultCode)
	if msg.ResultCode == protocal.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = meta.TransactionExceptionCode(exceptionCode)

	msg.BranchID, readN, _ = r.ReadInt64()
	totalReadN += readN

	return msg, totalReadN
}

func BranchReportRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.BranchReportRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.XID, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	msg.BranchID, _, _ = r.ReadInt64()
	branchStatus, _ := r.ReadByte()
	msg.Status = meta.BranchStatus(branchStatus)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceID, readN, _ = r.ReadString(int(length16))
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
	msg.BranchType = meta.BranchType(branchType)

	return msg, totalReadN
}

func BranchReportResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractTransactionResponseDecoder(in)
	abstractTransactionResponse := resp.(protocal.AbstractTransactionResponse)
	msg := protocal.BranchReportResponse{AbstractTransactionResponse: abstractTransactionResponse}
	return msg, totalReadN
}

func BranchRollbackRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractBranchEndRequestDecoder(in)
	abstractBranchEndRequest := req.(protocal.AbstractBranchEndRequest)
	msg := protocal.BranchRollbackRequest{AbstractBranchEndRequest: abstractBranchEndRequest}
	return msg, totalReadN
}

func BranchRollbackResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractBranchEndResponseDecoder(in)
	abstractBranchEndResponse := resp.(protocal.AbstractBranchEndResponse)
	msg := protocal.BranchRollbackResponse{AbstractBranchEndResponse: abstractBranchEndResponse}
	return msg, totalReadN
}

func GlobalBeginRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.GlobalBeginRequest{}

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
	msg := protocal.GlobalBeginResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocal.ResultCode(resultCode)
	if msg.ResultCode == protocal.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = meta.TransactionExceptionCode(exceptionCode)

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
	abstractGlobalEndRequest := req.(protocal.AbstractGlobalEndRequest)
	msg := protocal.GlobalCommitRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalCommitResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocal.AbstractGlobalEndResponse)
	msg := protocal.GlobalCommitResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalLockQueryRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := BranchRegisterRequestDecoder(in)
	branchRegisterRequest := req.(protocal.BranchRegisterRequest)
	msg := protocal.GlobalLockQueryRequest{BranchRegisterRequest: branchRegisterRequest}
	return msg, totalReadN
}

func GlobalLockQueryResponseDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.GlobalLockQueryResponse{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	resultCode, _ := r.ReadByte()
	totalReadN += 1
	msg.ResultCode = protocal.ResultCode(resultCode)
	if msg.ResultCode == protocal.ResultCodeFailed {
		length16, readN, _ = r.ReadUint16()
		totalReadN += readN
		if length16 > 0 {
			msg.Msg, readN, _ = r.ReadString(int(length16))
			totalReadN += readN
		}
	}

	exceptionCode, _ := r.ReadByte()
	totalReadN += 1
	msg.TransactionExceptionCode = meta.TransactionExceptionCode(exceptionCode)

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
	msg := protocal.GlobalReportRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.XID, readN, _ = r.ReadString(int(length16))
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
	msg.GlobalStatus = meta.GlobalStatus(globalStatus)

	return msg, totalReadN
}

func GlobalReportResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocal.AbstractGlobalEndResponse)
	msg := protocal.GlobalReportResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalRollbackRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractGlobalEndRequestDecoder(in)
	abstractGlobalEndRequest := req.(protocal.AbstractGlobalEndRequest)
	msg := protocal.GlobalRollbackRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalRollbackResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocal.AbstractGlobalEndResponse)
	msg := protocal.GlobalRollbackResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func GlobalStatusRequestDecoder(in []byte) (interface{}, int) {
	req, totalReadN := AbstractGlobalEndRequestDecoder(in)
	abstractGlobalEndRequest := req.(protocal.AbstractGlobalEndRequest)
	msg := protocal.GlobalStatusRequest{AbstractGlobalEndRequest: abstractGlobalEndRequest}
	return msg, totalReadN
}

func GlobalStatusResponseDecoder(in []byte) (interface{}, int) {
	resp, totalReadN := AbstractGlobalEndResponseDecoder(in)
	abstractGlobalEndResponse := resp.(protocal.AbstractGlobalEndResponse)
	msg := protocal.GlobalStatusResponse{AbstractGlobalEndResponse: abstractGlobalEndResponse}
	return msg, totalReadN
}

func UndoLogDeleteRequestDecoder(in []byte) (interface{}, int) {
	var (
		length16   uint16 = 0
		readN             = 0
		totalReadN        = 0
	)
	msg := protocal.UndoLogDeleteRequest{}

	r := byteio.BigEndianReader{Reader: bytes.NewReader(in)}
	branchType, _ := r.ReadByte()
	totalReadN += 1
	msg.BranchType = meta.BranchType(branchType)

	length16, readN, _ = r.ReadUint16()
	totalReadN += readN
	if length16 > 0 {
		msg.ResourceID, readN, _ = r.ReadString(int(length16))
		totalReadN += readN
	}

	msg.SaveDays, readN, _ = r.ReadInt16()
	totalReadN += readN

	return msg, totalReadN
}
