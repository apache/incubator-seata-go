package error

import (
	"github.com/pkg/errors"
)

type ErrorCode int32

const (
	ErrorCode_IllegalState ErrorCode = 40001
)

var (
	Error_TooManySessions  = errors.New("too many seeessions")
	Error_HeartBeatTimeOut = errors.New("heart beat time out")
)
