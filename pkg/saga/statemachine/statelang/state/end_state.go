package state

import (
	"seata.apache.org/seata-go/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
)

type EndState interface {
	statelang.State
}

type SucceedEndState interface {
	EndState
}

type SucceedEndStateImpl struct {
	*statelang.BaseState
}

func NewSucceedEndStateImpl() *SucceedEndStateImpl {
	s := &SucceedEndStateImpl{
		BaseState: statelang.NewBaseState(),
	}
	s.SetType(constant.StateTypeSucceed)
	return s
}

type FailEndState interface {
	EndState

	ErrorCode() string

	Message() string
}

type FailEndStateImpl struct {
	*statelang.BaseState
	errorCode string
	message   string
}

func NewFailEndStateImpl() *FailEndStateImpl {
	s := &FailEndStateImpl{
		BaseState: statelang.NewBaseState(),
	}
	s.SetType(constant.StateTypeFail)
	return s
}

func (f *FailEndStateImpl) ErrorCode() string {
	return f.errorCode
}

func (f *FailEndStateImpl) SetErrorCode(errorCode string) {
	f.errorCode = errorCode
}

func (f *FailEndStateImpl) Message() string {
	return f.message
}

func (f *FailEndStateImpl) SetMessage(message string) {
	f.message = message
}
