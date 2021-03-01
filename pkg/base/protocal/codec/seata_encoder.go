package codec

import (
	"bytes"
)

import (
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/protocal"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

func AbstractResultMessageEncoder(in interface{}) []byte {
	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	message := in.(protocal.AbstractResultMessage)

	w.WriteByte(byte(message.ResultCode))
	if message.ResultCode == protocal.ResultCodeFailed {
		var msg string
		if message.Msg != "" {
			if len(message.Msg) > 128 {
				msg = message.Msg[:128]
			} else {
				msg = message.Msg
			}
			// 暂时不考虑 message.Msg 包含中文的情况，这样字符串的长度就是 byte 数组的长度

			w.WriteInt16(int16(len(msg)))
			w.WriteString(msg)
		} else {
			w.WriteInt16(zero16)
		}
	}

	return b.Bytes()
}

func MergedWarpMessageEncoder(in interface{}) []byte {
	var (
		b      bytes.Buffer
		result = make([]byte, 0)
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.MergedWarpMessage)
	w.WriteInt16(int16(len(req.Msgs)))

	for i := 0; i < len(req.Msgs); i++ {
		msg := req.Msgs[i]
		encoder := getMessageEncoder(msg.GetTypeCode())
		if encoder != nil {
			data := encoder(msg)
			w.WriteInt16(msg.GetTypeCode())
			w.Write(data)
		}
	}

	size := uint32(b.Len())
	result = append(result, []byte{byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size)}...)
	result = append(result, b.Bytes()...)

	if len(req.Msgs) > 20 {
		log.Debugf("msg in one packet: %s ,buffer size: %s", len(req.Msgs), size)
	}
	return result
}

func MergeResultMessageEncoder(in interface{}) []byte {
	var (
		b      bytes.Buffer
		result = make([]byte, 0)
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.MergeResultMessage)
	w.WriteInt16(int16(len(req.Msgs)))

	for i := 0; i < len(req.Msgs); i++ {
		msg := req.Msgs[i]
		encoder := getMessageEncoder(msg.GetTypeCode())
		if encoder != nil {
			data := encoder(msg)
			w.WriteInt16(msg.GetTypeCode())
			w.Write(data)
		}
	}

	size := uint32(b.Len())
	result = append(result, []byte{byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size)}...)
	result = append(result, b.Bytes()...)

	if len(req.Msgs) > 20 {
		log.Debugf("msg in one packet: %s ,buffer size: %s", len(req.Msgs), size)
	}
	return result
}

func AbstractIdentifyRequestEncoder(in interface{}) []byte {
	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req := in.(protocal.AbstractIdentifyRequest)

	if req.Version != "" {
		w.WriteInt16(int16(len(req.Version)))
		w.WriteString(req.Version)
	} else {
		w.WriteInt16(zero16)
	}

	if req.ApplicationID != "" {
		w.WriteInt16(int16(len(req.ApplicationID)))
		w.WriteString(req.ApplicationID)
	} else {
		w.WriteInt16(zero16)
	}

	if req.TransactionServiceGroup != "" {
		w.WriteInt16(int16(len(req.TransactionServiceGroup)))
		w.WriteString(req.TransactionServiceGroup)
	} else {
		w.WriteInt16(zero16)
	}

	if req.ExtraData != nil {
		w.WriteUint16(uint16(len(req.ExtraData)))
		w.Write(req.ExtraData)
	} else {
		w.WriteInt16(zero16)
	}

	return b.Bytes()
}

func AbstractIdentifyResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.AbstractIdentifyResponse)

	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	if resp.Identified {
		w.WriteByte(byte(1))
	} else {
		w.WriteByte(byte(0))
	}

	if resp.Version != "" {
		w.WriteInt16(int16(len(resp.Version)))
		w.WriteString(resp.Version)
	} else {
		w.WriteInt16(zero16)
	}

	return b.Bytes()
}

func RegisterRMRequestEncoder(in interface{}) []byte {
	req := in.(protocal.RegisterRMRequest)
	data := AbstractIdentifyRequestEncoder(req.AbstractIdentifyRequest)

	var (
		zero32 int32 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	if req.ResourceIDs != "" {
		w.WriteInt32(int32(len(req.ResourceIDs)))
		w.WriteString(req.ResourceIDs)
	} else {
		w.WriteInt32(zero32)
	}

	result := append(data, b.Bytes()...)
	return result
}

func RegisterRMResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.RegisterRMResponse)
	return AbstractIdentifyResponseEncoder(resp.AbstractIdentifyResponse)
}

func RegisterTMRequestEncoder(in interface{}) []byte {
	req := in.(protocal.RegisterTMRequest)
	return AbstractIdentifyRequestEncoder(req.AbstractIdentifyRequest)
}

func RegisterTMResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.RegisterTMResponse)
	return AbstractIdentifyResponseEncoder(resp.AbstractIdentifyResponse)
}

func AbstractTransactionResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.AbstractTransactionResponse)
	data := AbstractResultMessageEncoder(resp.AbstractResultMessage)

	result := append(data, byte(resp.TransactionExceptionCode))

	return result
}

func AbstractBranchEndRequestEncoder(in interface{}) []byte {
	var (
		zero32 int32 = 0
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.AbstractBranchEndRequest)

	if req.XID != "" {
		w.WriteInt16(int16(len(req.XID)))
		w.WriteString(req.XID)
	} else {
		w.WriteInt16(zero16)
	}

	w.WriteInt64(req.BranchID)
	w.WriteByte(byte(req.BranchType))

	if req.ResourceID != "" {
		w.WriteInt16(int16(len(req.ResourceID)))
		w.WriteString(req.ResourceID)
	} else {
		w.WriteInt16(zero16)
	}

	if req.ApplicationData != nil {
		w.WriteUint32(uint32(len(req.ApplicationData)))
		w.Write(req.ApplicationData)
	} else {
		w.WriteInt32(zero32)
	}

	return b.Bytes()
}

func AbstractBranchEndResponseEncoder(in interface{}) []byte {
	resp, _ := in.(protocal.AbstractBranchEndResponse)
	data := AbstractTransactionResponseEncoder(resp.AbstractTransactionResponse)

	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	if resp.XID != "" {
		w.WriteInt16(int16(len(resp.XID)))
		w.WriteString(resp.XID)
	} else {
		w.WriteInt16(zero16)
	}

	w.WriteInt64(resp.BranchID)
	w.WriteByte(byte(resp.BranchStatus))

	result := append(data, b.Bytes()...)

	return result
}

func AbstractGlobalEndRequestEncoder(in interface{}) []byte {
	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.AbstractGlobalEndRequest)

	if req.XID != "" {
		w.WriteInt16(int16(len(req.XID)))
		w.WriteString(req.XID)
	} else {
		w.WriteInt16(zero16)
	}
	if req.ExtraData != nil {
		w.WriteUint16(uint16(len(req.ExtraData)))
		w.Write(req.ExtraData)
	} else {
		w.WriteInt16(zero16)
	}

	return b.Bytes()
}

func AbstractGlobalEndResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.AbstractGlobalEndResponse)
	data := AbstractTransactionResponseEncoder(resp.AbstractTransactionResponse)

	result := append(data, byte(resp.GlobalStatus))

	return result
}

func BranchCommitRequestEncoder(in interface{}) []byte {
	req := in.(protocal.BranchCommitRequest)
	return AbstractBranchEndRequestEncoder(req.AbstractBranchEndRequest)
}

func BranchCommitResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.BranchCommitResponse)
	return AbstractBranchEndResponseEncoder(resp.AbstractBranchEndResponse)
}

func BranchRegisterRequestEncoder(in interface{}) []byte {
	var (
		zero32 int32 = 0
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.BranchRegisterRequest)

	if req.XID != "" {
		w.WriteInt16(int16(len(req.XID)))
		w.WriteString(req.XID)
	} else {
		w.WriteInt16(zero16)
	}

	w.WriteByte(byte(req.BranchType))

	if req.ResourceID != "" {
		w.WriteInt16(int16(len(req.ResourceID)))
		w.WriteString(req.ResourceID)
	} else {
		w.WriteInt16(zero16)
	}

	if req.LockKey != "" {
		w.WriteInt32(int32(len(req.LockKey)))
		w.WriteString(req.LockKey)
	} else {
		w.WriteInt32(zero32)
	}

	if req.ApplicationData != nil {
		w.WriteUint32(uint32(len(req.ApplicationData)))
		w.Write(req.ApplicationData)
	} else {
		w.WriteInt32(zero32)
	}

	return b.Bytes()
}

func BranchRegisterResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.BranchRegisterResponse)
	data := AbstractTransactionResponseEncoder(resp.AbstractTransactionResponse)

	c := uint64(resp.BranchID)
	branchIDBytes := []byte{
		byte(c >> 56),
		byte(c >> 48),
		byte(c >> 40),
		byte(c >> 32),
		byte(c >> 24),
		byte(c >> 16),
		byte(c >> 8),
		byte(c),
	}
	result := append(data, branchIDBytes...)

	return result
}

func BranchReportRequestEncoder(in interface{}) []byte {
	var (
		zero32 int32 = 0
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.BranchReportRequest)

	if req.XID != "" {
		w.WriteInt16(int16(len(req.XID)))
		w.WriteString(req.XID)
	} else {
		w.WriteInt16(zero16)
	}

	w.WriteInt64(req.BranchID)
	w.WriteByte(byte(req.Status))

	if req.ResourceID != "" {
		w.WriteInt16(int16(len(req.ResourceID)))
		w.WriteString(req.ResourceID)
	} else {
		w.WriteInt16(zero16)
	}

	if req.ApplicationData != nil {
		w.WriteUint32(uint32(len(req.ApplicationData)))
		w.Write(req.ApplicationData)
	} else {
		w.WriteInt32(zero32)
	}

	w.WriteByte(byte(req.BranchType))

	return b.Bytes()
}

func BranchReportResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.BranchReportResponse)
	return AbstractTransactionResponseEncoder(resp.AbstractTransactionResponse)
}

func BranchRollbackRequestEncoder(in interface{}) []byte {
	req := in.(protocal.BranchRollbackRequest)
	return AbstractBranchEndRequestEncoder(req.AbstractBranchEndRequest)
}

func BranchRollbackResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.BranchRollbackResponse)
	return AbstractBranchEndResponseEncoder(resp.AbstractBranchEndResponse)
}

func GlobalBeginRequestEncoder(in interface{}) []byte {
	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.GlobalBeginRequest)

	w.WriteInt32(req.Timeout)
	if req.TransactionName != "" {
		w.WriteInt16(int16(len(req.TransactionName)))
		w.WriteString(req.TransactionName)
	} else {
		w.WriteInt16(zero16)
	}

	return b.Bytes()
}

func GlobalBeginResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.GlobalBeginResponse)
	data := AbstractTransactionResponseEncoder(resp.AbstractTransactionResponse)

	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	if resp.Xid != "" {
		w.WriteInt16(int16(len(resp.Xid)))
		w.WriteString(resp.Xid)
	} else {
		w.WriteInt16(zero16)
	}
	if resp.ExtraData != nil {
		w.WriteUint16(uint16(len(resp.ExtraData)))
		w.Write(resp.ExtraData)
	} else {
		w.WriteInt16(zero16)
	}

	result := append(data, b.Bytes()...)

	return result
}

func GlobalCommitRequestEncoder(in interface{}) []byte {
	req := in.(protocal.GlobalCommitRequest)
	return AbstractGlobalEndRequestEncoder(req.AbstractGlobalEndRequest)
}

func GlobalCommitResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.GlobalCommitResponse)
	return AbstractGlobalEndResponseEncoder(resp.AbstractGlobalEndResponse)
}

func GlobalLockQueryRequestEncoder(in interface{}) []byte {
	return BranchRegisterRequestEncoder(in)
}

func GlobalLockQueryResponseEncoder(in interface{}) []byte {
	resp, _ := in.(protocal.GlobalLockQueryResponse)
	data := AbstractTransactionResponseEncoder(resp.AbstractTransactionResponse)

	var result []byte
	if resp.Lockable {
		result = append(data, byte(0), byte(1))
	} else {
		result = append(data, byte(0), byte(0))
	}

	return result
}

func GlobalReportRequestEncoder(in interface{}) []byte {
	req, _ := in.(protocal.GlobalReportRequest)
	data := AbstractGlobalEndRequestEncoder(req.AbstractGlobalEndRequest)

	result := append(data, byte(req.GlobalStatus))
	return result
}

func GlobalReportResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.GlobalReportResponse)
	return AbstractGlobalEndResponseEncoder(resp.AbstractGlobalEndResponse)
}

func GlobalRollbackRequestEncoder(in interface{}) []byte {
	req := in.(protocal.GlobalRollbackRequest)
	return AbstractGlobalEndRequestEncoder(req.AbstractGlobalEndRequest)
}

func GlobalRollbackResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.GlobalRollbackResponse)
	return AbstractGlobalEndResponseEncoder(resp.AbstractGlobalEndResponse)
}

func GlobalStatusRequestEncoder(in interface{}) []byte {
	req := in.(protocal.GlobalStatusRequest)
	return AbstractGlobalEndRequestEncoder(req.AbstractGlobalEndRequest)
}

func GlobalStatusResponseEncoder(in interface{}) []byte {
	resp := in.(protocal.GlobalStatusResponse)
	return AbstractGlobalEndResponseEncoder(resp.AbstractGlobalEndResponse)
}

func UndoLogDeleteRequestEncoder(in interface{}) []byte {
	var (
		zero16 int16 = 0
		b      bytes.Buffer
	)
	w := byteio.BigEndianWriter{Writer: &b}

	req, _ := in.(protocal.UndoLogDeleteRequest)

	w.WriteByte(byte(req.BranchType))
	if req.ResourceID != "" {
		w.WriteInt16(int16(len(req.ResourceID)))
		w.WriteString(req.ResourceID)
	} else {
		w.WriteInt16(zero16)
	}
	w.WriteInt16(req.SaveDays)

	return b.Bytes()
}
