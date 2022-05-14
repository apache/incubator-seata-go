package protocol

import (
	model2 "github.com/seata/seata-go/pkg/common/model"
)

type AbstractTransactionResponse struct {
	AbstractResultMessage
	TransactionExceptionCode model2.TransactionExceptionCode
}

type AbstractBranchEndRequest struct {
	Xid             string
	BranchId        int64
	BranchType      model2.BranchType
	ResourceId      string
	ApplicationData []byte
}

type AbstractBranchEndResponse struct {
	AbstractTransactionResponse

	Xid          string
	BranchId     int64
	BranchStatus model2.BranchStatus
}

type AbstractGlobalEndRequest struct {
	Xid       string
	ExtraData []byte
}

type AbstractGlobalEndResponse struct {
	AbstractTransactionResponse

	GlobalStatus model2.GlobalStatus
}

type BranchRegisterRequest struct {
	Xid             string
	BranchType      model2.BranchType
	ResourceId      string
	LockKey         string
	ApplicationData []byte
}

func (req BranchRegisterRequest) GetTypeCode() MessageType {
	return MessageTypeBranchRegister
}

type BranchRegisterResponse struct {
	AbstractTransactionResponse

	BranchId int64
}

func (resp BranchRegisterResponse) GetTypeCode() MessageType {
	return MessageTypeBranchRegisterResult
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
	return MessageTypeBranchStatusReport
}

type BranchReportResponse struct {
	AbstractTransactionResponse
}

func (resp BranchReportResponse) GetTypeCode() MessageType {
	return MessageTypeBranchStatusReportResult
}

type BranchCommitRequest struct {
	AbstractBranchEndRequest
}

func (req BranchCommitRequest) GetTypeCode() MessageType {
	return MessageTypeBranchCommit
}

type BranchCommitResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchCommitResponse) GetTypeCode() MessageType {
	return MessageTypeBranchCommitResult
}

type BranchRollbackRequest struct {
	AbstractBranchEndRequest
}

func (req BranchRollbackRequest) GetTypeCode() MessageType {
	return MessageTypeBranchRollback
}

type BranchRollbackResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchRollbackResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalRollbackResult
}

type GlobalBeginRequest struct {
	Timeout         int32
	TransactionName string
}

func (req GlobalBeginRequest) GetTypeCode() MessageType {
	return MessageTypeGlobalBegin
}

type GlobalBeginResponse struct {
	AbstractTransactionResponse

	Xid       string
	ExtraData []byte
}

func (resp GlobalBeginResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalBeginResult
}

type GlobalStatusRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalStatusRequest) GetTypeCode() MessageType {
	return MessageTypeGlobalStatus
}

type GlobalStatusResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalStatusResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalStatusResult
}

type GlobalLockQueryRequest struct {
	BranchRegisterRequest
}

func (req GlobalLockQueryRequest) GetTypeCode() MessageType {
	return MessageTypeGlobalLockQuery
}

type GlobalLockQueryResponse struct {
	AbstractTransactionResponse

	Lockable bool
}

func (resp GlobalLockQueryResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalLockQueryResult
}

type GlobalReportRequest struct {
	AbstractGlobalEndRequest

	GlobalStatus model2.GlobalStatus
}

func (req GlobalReportRequest) GetTypeCode() MessageType {
	return MessageTypeGlobalStatus
}

type GlobalReportResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalReportResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalStatusResult
}

type GlobalCommitRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalCommitRequest) GetTypeCode() MessageType {
	return MessageTypeGlobalCommit
}

type GlobalCommitResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalCommitResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalCommitResult
}

type GlobalRollbackRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalRollbackRequest) GetTypeCode() MessageType {
	return MessageTypeGlobalRollback
}

type GlobalRollbackResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalRollbackResponse) GetTypeCode() MessageType {
	return MessageTypeGlobalRollbackResult
}

type UndoLogDeleteRequest struct {
	ResourceId string
	SaveDays   MessageType
	BranchType model2.BranchType
}

func (req UndoLogDeleteRequest) GetTypeCode() MessageType {
	return MessageTypeRmDeleteUndolog
}
