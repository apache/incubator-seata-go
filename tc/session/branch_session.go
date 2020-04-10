package session

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/dk-lockdown/seata-golang/logging"
	"github.com/dk-lockdown/seata-golang/meta"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/util"
	"vimagination.zapto.org/byteio"
)

type BranchSession struct{
	Xid string

	TransactionId int64

	BranchId int64

	ResourceGroupId string

	ResourceId string

	LockKey string

	BranchType meta.BranchType

	Status meta.BranchStatus

	ClientId string

	ApplicationData []byte
}

func NewBranchSession() *BranchSession {
	return &BranchSession{}
}

func NewBranchSessionByGlobal(gs GlobalSession,
								branchType meta.BranchType,
								resourceId string,
								applicationData []byte,
								lockKeys string,
								clientId string) *BranchSession {
	bs := NewBranchSession()
	bs.SetXid(gs.Xid)
	bs.SetTransactionId(gs.TransactionId)
	bs.SetBranchId(util.GeneratorUUID())
	bs.SetBranchType(branchType)
	bs.SetResourceId(resourceId)
	bs.SetLockKey(lockKeys)
	bs.SetClientId(clientId)
	bs.SetApplicationData(applicationData)
	return bs
}


func (bs *BranchSession) SetXid(xid string) *BranchSession {
	bs.Xid = xid
	return bs
}

func (bs *BranchSession) SetTransactionId(transactionId int64) *BranchSession {
	bs.TransactionId = transactionId
	return bs
}

func (bs *BranchSession) SetBranchId(branchId int64) *BranchSession {
	bs.BranchId = branchId
	return bs
}


func (bs *BranchSession) SetResourceGroupId(ResourceGroupId string) *BranchSession {
	bs.ResourceGroupId = ResourceGroupId
	return bs
}

func (bs *BranchSession) SetResourceId(resourceId string) *BranchSession {
	bs.ResourceId = resourceId
	return bs
}

func (bs *BranchSession) SetLockKey(lockKey string) *BranchSession {
	bs.LockKey = lockKey
	return bs
}

func (bs *BranchSession) SetBranchType(branchType meta.BranchType) *BranchSession {
	bs.BranchType = branchType
	return bs
}

func (bs *BranchSession) SetStatus(status meta.BranchStatus) *BranchSession {
	bs.Status = status
	return bs
}

func (bs *BranchSession) SetClientId(clientId string) *BranchSession {
	bs.ClientId = clientId
	return bs
}

func (bs *BranchSession) SetApplicationData(applicationData []byte) *BranchSession {
	bs.ApplicationData = applicationData
	return bs
}

func (bs *BranchSession) CompareTo(session *BranchSession) int {
	return int(bs.BranchId - session.BranchId)
}

func (bs *BranchSession) Encode() ([]byte, error) {
	var (
		zero32 int32 = 0
		zero16 int16 = 0
	)

	size := calBranchSessionSize(len(bs.ResourceId),len(bs.LockKey),len(bs.ClientId),len(bs.ApplicationData),len(bs.Xid))

	if size > config.GetStoreConfig().MaxBranchSessionSize {
		if bs.LockKey == "" {
			logging.Logger.Errorf("branch session size exceeded, size : %d maxBranchSessionSize : %d", size, config.GetStoreConfig().MaxBranchSessionSize)
			//todo compress
			return nil, errors.New("branch session size exceeded.")
		}
	}

	var b bytes.Buffer
	w := byteio.BigEndianWriter{Writer: &b}

	w.WriteInt64(bs.TransactionId)
	w.WriteInt64(bs.BranchId)
	if bs.ResourceId != "" {
		w.WriteUint32(uint32(len(bs.ResourceId)))
		w.WriteString( bs.ResourceId)
	} else {
		w.WriteInt32(zero32)
	}

	if bs.LockKey != "" {
		w.WriteUint32(uint32(len(bs.LockKey)))
		w.WriteString( bs.LockKey)
	} else {
		w.WriteInt32(zero32)
	}

	if bs.ClientId != "" {
		w.WriteUint16(uint16(len(bs.ClientId)))
		w.WriteString(bs.ClientId)
	} else {
		w.WriteInt16(zero16)
	}

	if bs.ApplicationData != nil {
		w.WriteUint32(uint32(len(bs.ApplicationData)))
		w.Write(bs.ApplicationData)
	} else {
		w.WriteInt32(zero32)
	}

	if bs.Xid != "" {
		w.WriteUint32(uint32(len(bs.Xid)))
		w.WriteString( bs.Xid)
	} else {
		w.WriteInt32(zero32)
	}

	w.WriteByte(byte(bs.BranchType))
	w.WriteByte(byte(bs.Status))

	return b.Bytes(), nil
}

func (bs *BranchSession) Decode(b []byte) {
	var length32 uint32 = 0
	var length16 uint16 = 0
	r := byteio.BigEndianReader{Reader:bytes.NewReader(b)}

	bs.TransactionId, _, _ = r.ReadInt64()
	bs.BranchId, _, _ = r.ReadInt64()

	length32, _, _ = r.ReadUint32()
	if length32 > 0 { bs.ResourceId, _, _ = r.ReadString(int(length32)) }

	length32, _, _ = r.ReadUint32()
	if length32 > 0 { bs.LockKey, _, _ = r.ReadString(int(length32)) }

	length16, _, _ = r.ReadUint16()
	if length16 > 0 { bs.ClientId, _, _ = r.ReadString(int(length16)) }

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.ApplicationData = make([]byte,int(length32))
		r.Read(bs.ApplicationData)
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 { bs.Xid, _, _ = r.ReadString(int(length32)) }

	branchType, _ := r.ReadByte()
	bs.BranchType = meta.BranchType(branchType)

	status, _ := r.ReadByte()
	bs.Status = meta.BranchStatus(status)
}

func calBranchSessionSize(resourceIdLen int,
	lockKeyLen int,
	clientIdLen int,
	applicationDataLen int,
	xidLen int) int{

	size := 8 + // transactionId
		8  + // branchId
		4  + // resourceIdBytes.length
		4  + // lockKeyBytes.length
		2  + // clientIdBytes.length
		4  + // applicationDataBytes.length
		4  + // xidBytes.size
		1  + // statusCode
		resourceIdLen +
		lockKeyLen +
		clientIdLen +
		applicationDataLen +
		xidLen +
		1

	return size
}
