package meta

import "errors"

type TransactionExceptionCode byte

const (
	/**
	 * Unknown transaction exception code.
	 */
	TransactionExceptionCodeUnknown TransactionExceptionCode = iota

	/**
	 * BeginFailed
	 */
	TransactionExceptionCodeBeginFailed

	/**
	 * Lock key conflict transaction exception code.
	 */
	TransactionExceptionCodeLockKeyConflict

	/**
	 * Io transaction exception code.
	 */
	IO

	/**
	 * Branch rollback failed retriable transaction exception code.
	 */
	TransactionExceptionCodeBranchRollbackFailedRetriable

	/**
	 * Branch rollback failed unretriable transaction exception code.
	 */
	TransactionExceptionCodeBranchRollbackFailedUnretriable

	/**
	 * Branch register failed transaction exception code.
	 */
	TransactionExceptionCodeBranchRegisterFailed

	/**
	 * Branch report failed transaction exception code.
	 */
	TransactionExceptionCodeBranchReportFailed

	/**
	 * Lockable check failed transaction exception code.
	 */
	TransactionExceptionCodeLockableCheckFailed

	/**
	 * Branch transaction not exist transaction exception code.
	 */
	TransactionExceptionCodeBranchTransactionNotExist

	/**
	 * Global transaction not exist transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionNotExist

	/**
	 * Global transaction not active transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionNotActive

	/**
	 * Global transaction status invalid transaction exception code.
	 */
	TransactionExceptionCodeGlobalTransactionStatusInvalid

	/**
	 * Failed to send branch commit request transaction exception code.
	 */
	TransactionExceptionCodeFailedToSendBranchCommitRequest

	/**
	 * Failed to send branch rollback request transaction exception code.
	 */
	TransactionExceptionCodeFailedToSendBranchRollbackRequest

	/**
	 * Failed to add branch transaction exception code.
	 */
	TransactionExceptionCodeFailedToAddBranch

	/**
	 * Failed to lock global transaction exception code.
	 */
	TransactionExceptionCodeFailedLockGlobalTranscation

	/**
	 * FailedWriteSession
	 */
	TransactionExceptionCodeFailedWriteSession

	/**
	 * Failed to holder exception code
	 */
	FailedStore
)

type TransactionException struct {
	Code    TransactionExceptionCode
	Message string
	Err     error
}

//Error 隐式继承 builtin.error 接口
func (e *TransactionException) Error() string {
	return "TransactionException: " + e.Message
}

func (e *TransactionException) Unwrap() error { return e.Err }

type TransactionExceptionOption func(exception *TransactionException)

func WithTransactionExceptionCode(code TransactionExceptionCode) TransactionExceptionOption {
	return func(exception *TransactionException) {
		exception.Code = code
	}
}

func WithMessage(message string) TransactionExceptionOption {
	return func(exception *TransactionException) {
		exception.Message = message
	}
}

func NewTransactionException(err error, opts ...TransactionExceptionOption) *TransactionException {
	var ex *TransactionException
	if errors.As(err, &ex) {
		return ex
	}
	ex = &TransactionException{
		Code:    TransactionExceptionCodeUnknown,
		Message: err.Error(),
		Err:     err,
	}
	for _, o := range opts {
		o(ex)
	}
	return ex
}
