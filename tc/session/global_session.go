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
	"github.com/dk-lockdown/seata-golang/base/common"
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/pkg/time"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/dk-lockdown/seata-golang/tc/config"
)

type GlobalSession struct {
	sync.Mutex

	Xid string

	TransactionId int64

	Status meta.GlobalStatus

	ApplicationId string

	TransactionServiceGroup string

	TransactionName string

	Timeout int32

	BeginTime int64

	ApplicationData []byte

	Active bool

	BranchSessions map[*BranchSession]bool
}

func NewGlobalSession() *GlobalSession {
	gs := &GlobalSession{
		BranchSessions: make(map[*BranchSession]bool),
	}
	gs.TransactionId = uuid.GeneratorUUID()
	gs.Xid = common.XID.GenerateXID(gs.TransactionId)
	gs.Active = true
	return gs
}

func (gs *GlobalSession) SetXid(xid string) *GlobalSession {
	gs.Xid = xid
	return gs
}

func (gs *GlobalSession) SetTransactionId(transactionId int64) *GlobalSession {
	gs.TransactionId = transactionId
	return gs
}

func (gs *GlobalSession) SetStatus(status meta.GlobalStatus) *GlobalSession {
	gs.Status = status
	return gs
}

func (gs *GlobalSession) SetApplicationId(applicationId string) *GlobalSession {
	gs.ApplicationId = applicationId
	return gs
}

func (gs *GlobalSession) SetTransactionServiceGroup(transactionServiceGroup string) *GlobalSession {
	gs.TransactionServiceGroup = transactionServiceGroup
	return gs
}

func (gs *GlobalSession) SetTransactionName(transactionName string) *GlobalSession {
	gs.TransactionName = transactionName
	return gs
}

func (gs *GlobalSession) SetTimeout(timeout int32) *GlobalSession {
	gs.Timeout = timeout
	return gs
}

func (gs *GlobalSession) SetBeginTime(beginTime int64) *GlobalSession {
	gs.BeginTime = beginTime
	return gs
}

func (gs *GlobalSession) SetApplicationData(applicationData []byte) *GlobalSession {
	gs.ApplicationData = applicationData
	return gs
}

func (gs *GlobalSession) SetActive(active bool) *GlobalSession {
	gs.Active = active
	return gs
}

func (gs *GlobalSession) Add(branchSession *BranchSession) {
	gs.Lock()
	defer gs.Unlock()
	branchSession.Status = meta.BranchStatusRegistered
	gs.BranchSessions[branchSession] = true
}

func (gs *GlobalSession) Remove(branchSession *BranchSession) {
	gs.Lock()
	defer gs.Unlock()
	delete(gs.BranchSessions, branchSession)
}

func (gs *GlobalSession) CanBeCommittedAsync() bool {
	gs.Lock()
	defer gs.Unlock()
	for branchSession := range gs.BranchSessions {
		if branchSession.BranchType == meta.BranchTypeTCC {
			return false
		}
	}
	return true
}

func (gs *GlobalSession) IsSaga() bool {
	gs.Lock()
	defer gs.Unlock()
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
	return (time.CurrentTimeMillis() - uint64(gs.BeginTime)) > uint64(2 * 6000)
}

func (gs *GlobalSession) GetSortedBranches() []*BranchSession {
	gs.Lock()
	defer gs.Unlock()
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

func (gs *GlobalSession) GetBranch(branchId int64) *BranchSession {
	gs.Lock()
	defer gs.Unlock()
	for branchSession := range gs.BranchSessions {
		if branchSession.BranchId == branchId {
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

	size := calGlobalSessionSize(len(gs.ApplicationId),len(gs.TransactionServiceGroup),len(gs.TransactionName),len(gs.Xid),len(gs.ApplicationData))

	if size > config.GetStoreConfig().MaxGlobalSessionSize {
		logging.Logger.Errorf("global session size exceeded, size : %d maxBranchSessionSize : %d", size, config.GetStoreConfig().MaxGlobalSessionSize)
		//todo compress
		return nil, errors.New("global session size exceeded.")
	}

	var b bytes.Buffer
	w := byteio.BigEndianWriter{Writer: &b}

	w.WriteInt64(gs.TransactionId)
	w.WriteInt32(gs.Timeout)

	// applicationId 长度不会超过 256
	if gs.ApplicationId != "" {
		w.WriteUint16(uint16(len(gs.ApplicationId)))
		w.WriteString(gs.ApplicationId)
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

	if gs.Xid != "" {
		w.WriteUint32(uint32(len(gs.Xid)))
		w.WriteString(gs.Xid)
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
	r := byteio.BigEndianReader{Reader:bytes.NewReader(b)}

	gs.TransactionId, _, _ = r.ReadInt64()
	gs.Timeout, _, _ = r.ReadInt32()

	length16, _, _ = r.ReadUint16()
	if length16 > 0 { gs.ApplicationId, _, _ = r.ReadString(int(length16)) }

	length16, _, _ = r.ReadUint16()
	if length16 > 0 { gs.TransactionServiceGroup, _, _ = r.ReadString(int(length16)) }

	length16, _, _ = r.ReadUint16()
	if length16 > 0 { gs.TransactionName, _, _ = r.ReadString(int(length16)) }

	length32, _, _ = r.ReadUint32()
	if length32 > 0 { gs.Xid, _, _ = r.ReadString(int(length32)) }

	length32, _, _ = r.ReadUint32()
	if length32 > 0 {
		gs.ApplicationData = make([]byte,length32,length32)
		r.Read(gs.ApplicationData)
	}

	gs.BeginTime, _, _ = r.ReadInt64()

	status, _ := r.ReadByte()
	gs.Status = meta.GlobalStatus(status)
}

func calGlobalSessionSize(	applicationIdLen int,
	serviceGroupLen int,
	txNameLen int,
	xidLen int,
	applicationDataLen int,
	) int{

	size := 8 + // transactionId
		4  + // timeout
		2  + // byApplicationIdBytes.length
		2  + // byServiceGroupBytes.length
		2  + // byTxNameBytes.length
		4  + // xidBytes.length
		4  + // applicationDataBytes.length
		8  + // beginTime
		1  + // statusCode
		applicationIdLen +
		serviceGroupLen +
		txNameLen +
		xidLen +
		applicationDataLen

	return size
}
