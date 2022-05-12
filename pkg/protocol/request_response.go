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

func (req BranchRegisterRequest) GetTypeCode() int16 {
	return TypeBranchRegister
}

type BranchRegisterResponse struct {
	AbstractTransactionResponse

	BranchId int64
}

func (resp BranchRegisterResponse) GetTypeCode() int16 {
	return TypeBranchRegisterResult
}

type BranchReportRequest struct {
	Xid             string
	BranchId        int64
	ResourceId      string
	Status          model2.BranchStatus
	ApplicationData []byte
	BranchType      model2.BranchType
}

func (req BranchReportRequest) GetTypeCode() int16 {
	return TypeBranchStatusReport
}

type BranchReportResponse struct {
	AbstractTransactionResponse
}

func (resp BranchReportResponse) GetTypeCode() int16 {
	return TypeBranchStatusReportResult
}

type BranchCommitRequest struct {
	AbstractBranchEndRequest
}

func (req BranchCommitRequest) GetTypeCode() int16 {
	return TypeBranchCommit
}

type BranchCommitResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchCommitResponse) GetTypeCode() int16 {
	return TypeBranchCommitResult
}

type BranchRollbackRequest struct {
	AbstractBranchEndRequest
}

func (req BranchRollbackRequest) GetTypeCode() int16 {
	return TypeBranchRollback
}

type BranchRollbackResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchRollbackResponse) GetTypeCode() int16 {
	return TypeGlobalRollbackResult
}

type GlobalBeginRequest struct {
	Timeout         int32
	TransactionName string
}

func (req GlobalBeginRequest) GetTypeCode() int16 {
	return TypeGlobalBegin
}

type GlobalBeginResponse struct {
	AbstractTransactionResponse

	Xid       string
	ExtraData []byte
}

func (resp GlobalBeginResponse) GetTypeCode() int16 {
	return TypeGlobalBeginResult
}

type GlobalStatusRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalStatusRequest) GetTypeCode() int16 {
	return TypeGlobalStatus
}

type GlobalStatusResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalStatusResponse) GetTypeCode() int16 {
	return TypeGlobalStatusResult
}

type GlobalLockQueryRequest struct {
	BranchRegisterRequest
}

func (req GlobalLockQueryRequest) GetTypeCode() int16 {
	return TypeGlobalLockQuery
}

type GlobalLockQueryResponse struct {
	AbstractTransactionResponse

	Lockable bool
}

func (resp GlobalLockQueryResponse) GetTypeCode() int16 {
	return TypeGlobalLockQueryResult
}

type GlobalReportRequest struct {
	AbstractGlobalEndRequest

	GlobalStatus model2.GlobalStatus
}

func (req GlobalReportRequest) GetTypeCode() int16 {
	return TypeGlobalStatus
}

type GlobalReportResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalReportResponse) GetTypeCode() int16 {
	return TypeGlobalStatusResult
}

type GlobalCommitRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalCommitRequest) GetTypeCode() int16 {
	return TypeGlobalCommit
}

type GlobalCommitResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalCommitResponse) GetTypeCode() int16 {
	return TypeGlobalCommitResult
}

type GlobalRollbackRequest struct {
	AbstractGlobalEndRequest
}

func (req GlobalRollbackRequest) GetTypeCode() int16 {
	return TypeGlobalRollback
}

type GlobalRollbackResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalRollbackResponse) GetTypeCode() int16 {
	return TypeGlobalRollbackResult
}

type UndoLogDeleteRequest struct {
	ResourceId string
	SaveDays   int16
	BranchType model2.BranchType
}

func (req UndoLogDeleteRequest) GetTypeCode() int16 {
	return TypeRmDeleteUndolog
}
