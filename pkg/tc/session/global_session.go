package session

import (
	"bytes"
	"sort"
	"sync"
)

import (
	"github.com/pkg/errors"
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/time"
	"github.com/transaction-wg/seata-golang/pkg/util/uuid"
)

type GlobalSession struct {
	sync.Mutex

	XID string

	TransactionID int64

	Status meta.GlobalStatus

	ApplicationID string

	TransactionServiceGroup string

	TransactionName string

	Timeout int32

	BeginTime int64

	ApplicationData []byte

	Active bool

	BranchSessions map[*BranchSession]bool
}

type GlobalSessionOption func(session *GlobalSession)

func WithGsXID(xid string) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.XID = xid
	}
}

func WithGsTransactionID(transactionID int64) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.TransactionID = transactionID
	}
}

func WithGsStatus(status meta.GlobalStatus) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.Status = status
	}
}

func WithGsApplicationID(applicationID string) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.ApplicationID = applicationID
	}
}

func WithGsTransactionServiceGroup(transactionServiceGroup string) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.TransactionServiceGroup = transactionServiceGroup
	}
}

func WithGsTransactionName(transactionName string) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.TransactionName = transactionName
	}
}

func WithGsTimeout(timeout int32) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.Timeout = timeout
	}
}

func WithGsBeginTime(beginTime int64) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.BeginTime = beginTime
	}
}

func WithGsApplicationData(applicationData []byte) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.ApplicationData = applicationData
	}
}

func WithGsActive(active bool) GlobalSessionOption {
	return func(session *GlobalSession) {
		session.Active = active
	}
}

func NewGlobalSession(opts ...GlobalSessionOption) *GlobalSession {
	gs := &GlobalSession{
		BranchSessions: make(map[*BranchSession]bool),
		TransactionID:  uuid.GeneratorUUID(),
		Active:         true,
	}
	gs.XID = common.GetXID().GenerateXID(gs.TransactionID)
	for _, o := range opts {
		o(gs)
	}
	return gs
}

func (gs *GlobalSession) Add(branchSession *BranchSession) {
	branchSession.Status = meta.BranchStatusRegistered
	gs.BranchSessions[branchSession] = true
}

func (gs *GlobalSession) Remove(branchSession *BranchSession) {
	delete(gs.BranchSessions, branchSession)
}

func (gs *GlobalSession) CanBeCommittedAsync() bool {
	for branchSession := range gs.BranchSessions {
		if branchSession.BranchType == meta.BranchTypeTCC {
			return false
		}
	}
	return true
}

func (gs *GlobalSession) IsSaga() bool {
	for branchSession := range gs.BranchSessions {
		if branchSession.BranchType == meta.BranchTypeSAGA {
			return true
		} else {
			return false
		}
	}
	return false
}

func (gs *GlobalSession) IsTimeout() bool {
	return (time.CurrentTimeMillis() - uint64(gs.BeginTime)) > uint64(gs.Timeout)
}

func (gs *GlobalSession) IsRollbackingDead() bool {
	return (time.CurrentTimeMillis() - uint64(gs.BeginTime)) > uint64(2*6000)
}

func (gs *GlobalSession) GetSortedBranches() []*BranchSession {
	var branchSessions = make([]*BranchSession, 0)

	for branchSession := range gs.BranchSessions {
		branchSessions = append(branchSessions, branchSession)
	}
	return branchSessions
}

func (gs *GlobalSession) GetReverseSortedBranches() []*BranchSession {
	var branchSessions = gs.GetSortedBranches()
	sort.Reverse(BranchSessionSlice(branchSessions))
	return branchSessions
}

type BranchSessionSlice []*BranchSession

func (p BranchSessionSlice) Len() int           { return len(p) }
func (p BranchSessionSlice) Less(i, j int) bool { return p[i].CompareTo(p[j]) > 0 }
func (p BranchSessionSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (gs *GlobalSession) GetBranch(branchID int64) *BranchSession {
	gs.Lock()
	defer gs.Unlock()
	for branchSession := range gs.BranchSessions {
		if branchSession.BranchID == branchID {
			return branchSession
		}
	}
	return nil
}

func (gs *GlobalSession) HasBranch() bool {
	return len(gs.BranchSessions) > 0
}

func (gs *GlobalSession) Begin() {
	gs.Status = meta.GlobalStatusBegin
	gs.BeginTime = int64(time.CurrentTimeMillis())
	gs.Active = true
}

func (gs *GlobalSession) Encode() ([]byte, error) {
	var (
		zero32 int32 = 0
		zero16 int16 = 0
	)

	size := calGlobalSessionSize(len(gs.ApplicationID), len(gs.TransactionServiceGroup), len(gs.TransactionName), len(gs.XID), len(gs.ApplicationData))

	if size > config.GetStoreConfig().MaxGlobalSessionSize {
		log.Errorf("global session size exceeded, size : %d maxGlobalSessionSize : %d", size, config.GetStoreConfig().MaxGlobalSessionSize)
		//todo compress
		return nil, errors.New("global session size exceeded.")
	}

	var b bytes.Buffer
	w := byteio.BigEndianWriter{Writer: &b}

	w.WriteInt64(gs.TransactionID)
	w.WriteInt32(gs.Timeout)

	// applicationID 长度不会超过 256
	if gs.ApplicationID != "" {
		w.WriteUint16(uint16(len(gs.ApplicationID)))
		w.WriteString(gs.ApplicationID)
	} else {
		w.WriteInt16(zero16)
	}

	if gs.TransactionServiceGroup != "" {
		w.WriteUint16(uint16(len(gs.TransactionServiceGroup)))
		w.WriteString(gs.TransactionServiceGroup)
	} else {
		w.WriteInt16(zero16)
	}

	if gs.TransactionName != "" {
		w.WriteUint16(uint16(len(gs.TransactionName)))
		w.WriteString(gs.TransactionName)
	} else {
		w.WriteInt16(zero16)
	}

	if gs.XID != "" {
		w.WriteUint32(uint32(len(gs.XID)))
		w.WriteString(gs.XID)
	} else {
		w.WriteInt32(zero32)
	}

	if gs.ApplicationData != nil {
		w.WriteUint32(uint32(len(gs.ApplicationData)))
		w.Write(gs.ApplicationData)
	} else {
		w.WriteInt32(zero32)
	}

	w.WriteInt64(gs.BeginTime)
	w.WriteByte(byte(gs.Status))

	return b.Bytes(), nil
}

func (gs *GlobalSession) Decode(b []byte) {
	var length32 uint32 = 0
	var length16 uint16 = 0
	r := byteio.BigEndianReader{Reader: bytes.NewReader(b)}

	gs.TransactionID, _, _ = r.ReadInt64()
	gs.Timeout, _, _ = r.ReadInt32()

	length16, _, _ = r.ReadUint16()
	if length16 > 0 {
		gs.ApplicationID, _, _ = r.ReadString(int(length16))
	}

	length16, _, _ = r.ReadUint16()
	if length16 > 0 {
		gs.TransactionServiceGroup, _, _ = r.ReadString(int(length16))
	}

	length16, _, _ = r.ReadUint16()
	if length16 > 0 {
		gs.TransactionName, _, _ = r.ReadString(int(length16))
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		gs.XID, _, _ = r.ReadString(int(length32))
	}

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		gs.ApplicationData = make([]byte, length32, length32)
		r.Read(gs.ApplicationData)
	}

	gs.BeginTime, _, _ = r.ReadInt64()

	status, _ := r.ReadByte()
	gs.Status = meta.GlobalStatus(status)
}

func calGlobalSessionSize(applicationIDLen int,
	serviceGroupLen int,
	txNameLen int,
	xidLen int,
	applicationDataLen int,
) int {

	size := 8 + // transactionID
		4 + // timeout
		2 + // byApplicationIDBytes.length
		2 + // byServiceGroupBytes.length
		2 + // byTxNameBytes.length
		4 + // xidBytes.length
		4 + // applicationDataBytes.length
		8 + // beginTime
		1 + // statusCode
		applicationIDLen +
		serviceGroupLen +
		txNameLen +
		xidLen +
		applicationDataLen

	return size
}
