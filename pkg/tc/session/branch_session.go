package session

import (
	"bytes"
)

import (
	"github.com/pkg/errors"

	"vimagination.zapto.org/byteio"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/uuid"
)

type BranchSession struct {
	XID string

	TransactionID int64

	BranchID int64

	ResourceGroupID string

	ResourceID string

	LockKey string

	BranchType meta.BranchType

	Status meta.BranchStatus

	ClientID string

	ApplicationData []byte
}

type BranchSessionOption func(session *BranchSession)

func WithBsXid(xid string) BranchSessionOption {
	return func(session *BranchSession) {
		session.XID = xid
	}
}

func WithBsTransactionID(transactionID int64) BranchSessionOption {
	return func(session *BranchSession) {
		session.TransactionID = transactionID
	}
}

func WithBsBranchID(branchID int64) BranchSessionOption {
	return func(session *BranchSession) {
		session.BranchID = branchID
	}
}

func WithBsResourceGroupID(resourceGroupID string) BranchSessionOption {
	return func(session *BranchSession) {
		session.ResourceGroupID = resourceGroupID
	}
}

func WithBsResourceID(resourceID string) BranchSessionOption {
	return func(session *BranchSession) {
		session.ResourceID = resourceID
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

func WithBsClientID(clientID string) BranchSessionOption {
	return func(session *BranchSession) {
		session.ClientID = clientID
	}
}

func WithBsApplicationData(applicationData []byte) BranchSessionOption {
	return func(session *BranchSession) {
		session.ApplicationData = applicationData
	}
}

func NewBranchSession(opts ...BranchSessionOption) *BranchSession {
	session := &BranchSession{
		BranchID: uuid.NextID(),
		Status:   meta.BranchStatusUnknown,
	}
	for _, o := range opts {
		o(session)
	}
	return session
}

func NewBranchSessionByGlobal(gs GlobalSession, opts ...BranchSessionOption) *BranchSession {
	bs := &BranchSession{
		XID:           gs.XID,
		TransactionID: gs.TransactionID,
		BranchID:      uuid.NextID(),
		Status:        meta.BranchStatusUnknown,
	}
	for _, o := range opts {
		o(bs)
	}
	return bs
}

func (bs *BranchSession) CompareTo(session *BranchSession) int {
	return int(bs.BranchID - session.BranchID)
}

func (bs *BranchSession) Encode() ([]byte, error) {
	var (
		zero32 int32 = 0
		zero16 int16 = 0
	)

	size := calBranchSessionSize(len(bs.ResourceID), len(bs.LockKey), len(bs.ClientID), len(bs.ApplicationData), len(bs.XID))

	if size > config.GetStoreConfig().MaxBranchSessionSize {
		if bs.LockKey == "" {
			log.Errorf("branch session size exceeded, size : %d maxBranchSessionSize : %d", size, config.GetStoreConfig().MaxBranchSessionSize)
			//todo compress
			return nil, errors.New("branch session size exceeded.")
		}
	}

	var b bytes.Buffer
	w := byteio.BigEndianWriter{Writer: &b}

	w.WriteInt64(bs.TransactionID)
	w.WriteInt64(bs.BranchID)
	if bs.ResourceID != "" {
		w.WriteUint32(uint32(len(bs.ResourceID)))
		w.WriteString(bs.ResourceID)
	} else {
		w.WriteInt32(zero32)
	}

	if bs.LockKey != "" {
		w.WriteUint32(uint32(len(bs.LockKey)))
		w.WriteString(bs.LockKey)
	} else {
		w.WriteInt32(zero32)
	}

	if bs.ClientID != "" {
		w.WriteUint16(uint16(len(bs.ClientID)))
		w.WriteString(bs.ClientID)
	} else {
		w.WriteInt16(zero16)
	}

	if bs.ApplicationData != nil {
		w.WriteUint32(uint32(len(bs.ApplicationData)))
		w.Write(bs.ApplicationData)
	} else {
		w.WriteInt32(zero32)
	}

	if bs.XID != "" {
		w.WriteUint32(uint32(len(bs.XID)))
		w.WriteString(bs.XID)
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

	bs.TransactionID, _, _ = r.ReadInt64()
	bs.BranchID, _, _ = r.ReadInt64()

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.ResourceID, _, _ = r.ReadString(int(length32))
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.LockKey, _, _ = r.ReadString(int(length32))
	}

	length16, _, _ = r.ReadUint16()
	if length16 > 0 {
		bs.ClientID, _, _ = r.ReadString(int(length16))
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.ApplicationData = make([]byte, int(length32))
		r.Read(bs.ApplicationData)
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		bs.XID, _, _ = r.ReadString(int(length32))
	}

	branchType, _ := r.ReadByte()
	bs.BranchType = meta.BranchType(branchType)

	status, _ := r.ReadByte()
	bs.Status = meta.BranchStatus(status)
}

func calBranchSessionSize(resourceIDLen int,
	lockKeyLen int,
	clientIDLen int,
	applicationDataLen int,
	xidLen int) int {

	size := 8 + // transactionID
		8 + // branchID
		4 + // resourceIDBytes.length
		4 + // lockKeyBytes.length
		2 + // clientIDBytes.length
		4 + // applicationDataBytes.length
		4 + // xidBytes.size
		1 + // statusCode
		resourceIDLen +
		lockKeyLen +
		clientIDLen +
		applicationDataLen +
		xidLen +
		1

	return size
}
