package message

import (
	model2 "github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/transaction"
)

type AbstractBranchEndRequest struct {
	MessageTypeAware
	Xid             string
	BranchId        int64
	BranchType      model2.BranchType
	ResourceId      string
	ApplicationData []byte
}

type AbstractGlobalEndRequest struct {
	Xid       string
	ExtraData []byte
}

type BranchRegisterRequest struct {
	Xid             string
	BranchType      model2.BranchType
	ResourceId      string
	LockKey         string
	ApplicationData []byte
}

func (req BranchRegisterRequest) GetTypeCode() MessageType {
	return MessageType_BranchRegister
}

type BranchReportRequest struct {
	Xid             string
	BranchId        int64
	ResourceId      string
	Status          model2.BranchStatus
	ApplicationData []byte
	BranchType      model2.BranchType
}

func (req BranchReportRequest) GetTypeCode() MessageType {
	return MessageType_BranchStatusReport
}

type BranchCommitRequest struct {
	AbstractBranchEndRequest
}

func (req BranchCommitRequest) GetTypeCode() MessageType {
	return MessageType_BranchCommit
}

type BranchRollbackRequest struct {
	AbstractBranchEndRequest
}

func (req BranchRollbackRequest) GetTypeCode() MessageType {
	return MessageType_BranchRollback
}

type GlobalBeginRequest struct {
	Timeout         int32
	TransactionName string
}

func (req GlobalBeginRequest) GetTypeCode() MessageType {
	return MessageType_GlobalBegin
}

type GlobalStatusRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalStatusRequest) GetTypeCode() MessageType {
	return MessageType_GlobalStatus
}

type GlobalLockQueryRequest struct {
	BranchRegisterRequest
}

func (req GlobalLockQueryRequest) GetTypeCode() MessageType {
	return MessageType_GlobalLockQuery
}

type GlobalReportRequest struct {
	AbstractGlobalEndRequest

	GlobalStatus transaction.GlobalStatus
}

func (req GlobalReportRequest) GetTypeCode() MessageType {
	return MessageType_GlobalStatus
}

type GlobalCommitRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalCommitRequest) GetTypeCode() MessageType {
	return MessageType_GlobalCommit
}

type GlobalRollbackRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalRollbackRequest) GetTypeCode() MessageType {
	return MessageType_GlobalRollback
}

type UndoLogDeleteRequest struct {
	ResourceId string
	SaveDays   MessageType
	BranchType model2.BranchType
}

func (req UndoLogDeleteRequest) GetTypeCode() MessageType {
	return MessageType_RmDeleteUndolog
}

type RegisterTMRequest struct {
	AbstractIdentifyRequest
}

func (req RegisterTMRequest) GetTypeCode() MessageType {
	return MessageType_RegClt
}

type RegisterTMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterTMResponse) GetTypeCode() MessageType {
	return MessageType_RegCltResult
}

type RegisterRMRequest struct {
	AbstractIdentifyRequest
	ResourceIds string
}

func (req RegisterRMRequest) GetTypeCode() MessageType {
	return MessageType_RegRm
}

type RegisterRMResponse struct {
	AbstractIdentifyResponse
}

func (resp RegisterRMResponse) GetTypeCode() MessageType {
	return MessageType_RegRmResult
}
