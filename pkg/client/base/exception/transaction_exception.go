package exception

import (
	"errors"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
)

// TransactionException
type TransactionException struct {
	Code    apis.ExceptionCode
	Message string
	Err     error
}

// Error
func (e *TransactionException) Error() string {
	return "TransactionException: " + e.Message
}

// Unwrap
func (e *TransactionException) Unwrap() error { return e.Err }

// TransactionExceptionOption for edit TransactionException
type TransactionExceptionOption func(exception *TransactionException)

// WithExceptionCode edit exception code in TransactionException
func WithExceptionCode(code apis.ExceptionCode) TransactionExceptionOption {
	return func(exception *TransactionException) {
		exception.Code = code
	}
}

// WithMessage edit message in TransactionException
func WithMessage(message string) TransactionExceptionOption {
	return func(exception *TransactionException) {
		exception.Message = message
	}
}

// NewTransactionException return a pointer to TransactionException
func NewTransactionException(err error, opts ...TransactionExceptionOption) *TransactionException {
	var ex *TransactionException
	if errors.As(err, &ex) {
		return ex
	}
	ex = &TransactionException{
		Code:    apis.UnknownErr,
		Message: err.Error(),
		Err:     err,
	}
	for _, o := range opts {
		o(ex)
	}
	return ex
}
