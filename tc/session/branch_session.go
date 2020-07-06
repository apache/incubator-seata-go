package session

import (
	"bytes"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
)

import (
	"github.com/pkg/errors"
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/config"
)

type BranchSession struct {
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

type BranchSessionOption func(session *BranchSession)

func WithBsXid(xid string) BranchSessionOption {
	return func(session *BranchSession) {
		session.Xid = xid
	}
}

func WithBsTransactionId(transactionId int64) BranchSessionOption {
	return func(session *BranchSession) {
		session.TransactionId = transactionId
	}
}

func WithBsBranchId(branchId int64) BranchSessionOption {
	return func(session *BranchSession) {
		session.BranchId = branchId
	}
}

func WithBsResourceGroupId(resourceGroupId string) BranchSessionOption {
	return func(session *BranchSession) {
		session.ResourceGroupId = resourceGroupId
	}
}

func WithBsResourceId(resourceId string) BranchSessionOption {
	return func(session *BranchSession) {
		session.ResourceId = resourceId
	}
}

func WithBsLockKey(lockKey string) BranchSessionOption {
	return func(session *BranchSession) {
		session.LockKey = lockKey
	}
}

func WithBsBranchType(branchType meta.BranchType) BranchSessionOption {
	return func(session *BranchSession) {
		session.BranchType = branchType
	}
}

func WithBsStatus(status meta.BranchStatus) BranchSessionOption {
	return func(session *BranchSession) {
		session.Status = status
	}
}

func WithBsClientId(clientId string) BranchSessionOption {
	return func(session *BranchSession) {
		session.ClientId = clientId
	}
}

func WithBsApplicationData(applicationData []byte) BranchSessionOption {
	return func(session *BranchSession) {
		session.ApplicationData = applicationData
	}
}

func NewBranchSession(opts ...BranchSessionOption) *BranchSession {
	session := &BranchSession{
		BranchId: uuid.GeneratorUUID(),
		Status:   meta.BranchStatusUnknown,
	}
	for _, o := range opts {
		o(session)
	}
	return session
}

func NewBranchSessionByGlobal(gs GlobalSession, opts ...BranchSessionOption) *BranchSession {
	bs := &BranchSession{
		Xid:           gs.Xid,
		TransactionId: gs.TransactionId,
		BranchId:      uuid.GeneratorUUID(),
		Status:        meta.BranchStatusUnknown,
	}
	for _, o := range opts {
		o(bs)
	}
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

	size := calBranchSessionSize(len(bs.ResourceId), len(bs.LockKey), len(bs.ClientId), len(bs.ApplicationData), len(bs.Xid))

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
		w.WriteString(bs.ResourceId)
	} else {
		w.WriteInt32(zero32)
	}

	if bs.LockKey != "" {
		w.WriteUint32(uint32(len(bs.LockKey)))
		w.WriteString(bs.LockKey)
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
		w.WriteString(bs.Xid)
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
	r := byteio.BigEndianReader{Reader: bytes.NewReader(b)}

	bs.TransactionId, _, _ = r.ReadInt64()
	bs.BranchId, _, _ = r.ReadInt64()

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.ResourceId, _, _ = r.ReadString(int(length32))
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.LockKey, _, _ = r.ReadString(int(length32))
	}

	length16, _, _ = r.ReadUint16()
	if length16 > 0 {
		bs.ClientId, _, _ = r.ReadString(int(length16))
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.ApplicationData = make([]byte, int(length32))
		r.Read(bs.ApplicationData)
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.Xid, _, _ = r.ReadString(int(length32))
	}

	branchType, _ := r.ReadByte()
	bs.BranchType = meta.BranchType(branchType)

	status, _ := r.ReadByte()
	bs.Status = meta.BranchStatus(status)
}

func calBranchSessionSize(resourceIdLen int,
	lockKeyLen int,
	clientIdLen int,
	applicationDataLen int,
	xidLen int) int {

	size := 8 + // transactionId
		8 + // branchId
		4 + // resourceIdBytes.length
		4 + // lockKeyBytes.length
		2 + // clientIdBytes.length
		4 + // applicationDataBytes.length
		4 + // xidBytes.size
		1 + // statusCode
		resourceIdLen +
		lockKeyLen +
		clientIdLen +
		applicationDataLen +
		xidLen +
		1

	return size
}
