package message

import (
	model2 "github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/transaction"
)

type AbstractTransactionResponse struct {
	AbstractResultMessage
	TransactionExceptionCode transaction.TransactionExceptionCode
}

type AbstractBranchEndResponse struct {
	AbstractTransactionResponse
	Xid          string
	BranchId     int64
	BranchStatus model2.BranchStatus
}

type AbstractGlobalEndResponse struct {
	AbstractTransactionResponse
	GlobalStatus transaction.GlobalStatus
}

type BranchRegisterResponse struct {
	AbstractTransactionResponse
	BranchId int64
}

func (resp BranchRegisterResponse) GetTypeCode() MessageType {
	return MessageType_BranchRegisterResult
}

type BranchReportResponse struct {
	AbstractTransactionResponse
}

func (resp BranchReportResponse) GetTypeCode() MessageType {
	return MessageType_BranchStatusReportResult
}

type BranchCommitResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchCommitResponse) GetTypeCode() MessageType {
	return MessageType_BranchCommitResult
}

type BranchRollbackResponse struct {
	AbstractBranchEndResponse
}

func (resp BranchRollbackResponse) GetTypeCode() MessageType {
	return MessageType_GlobalRollbackResult
}

type GlobalBeginResponse struct {
	AbstractTransactionResponse

	Xid       string
	ExtraData []byte
}

func (resp GlobalBeginResponse) GetTypeCode() MessageType {
	return MessageType_GlobalBeginResult
}

type GlobalStatusResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalStatusResponse) GetTypeCode() MessageType {
	return MessageType_GlobalStatusResult
}

type GlobalLockQueryResponse struct {
	AbstractTransactionResponse

	Lockable bool
}

func (resp GlobalLockQueryResponse) GetTypeCode() MessageType {
	return MessageType_GlobalLockQueryResult
}

type GlobalReportResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalReportResponse) GetTypeCode() MessageType {
	return MessageType_GlobalStatusResult
}

type GlobalCommitResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalCommitResponse) GetTypeCode() MessageType {
	return MessageType_GlobalCommitResult
}

type GlobalRollbackResponse struct {
	AbstractGlobalEndResponse
}

func (resp GlobalRollbackResponse) GetTypeCode() MessageType {
	return MessageType_GlobalRollbackResult
}
