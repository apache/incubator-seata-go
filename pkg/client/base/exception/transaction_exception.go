package exception

import (
	"errors"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
)

type TransactionException struct {
	Code    apis.ExceptionCode
	Message string
	Err     error
}

//Error 隐式继承 builtin.error 接口
func (e *TransactionException) Error() string {
	return "TransactionException: " + e.Message
}

func (e *TransactionException) Unwrap() error { return e.Err }

type TransactionExceptionOption func(exception *TransactionException)

func WithExceptionCode(code apis.ExceptionCode) TransactionExceptionOption {
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
		Code:    apis.UnknownErr,
		Message: err.Error(),
		Err:     err,
	}
	for _, o := range opts {
		o(ex)
	}
	return ex
}
