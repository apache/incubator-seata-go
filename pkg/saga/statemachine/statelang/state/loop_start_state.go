package state

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type LoopStartState interface {
	statelang.State
}

type LoopStartStateImpl struct {
	*statelang.BaseState
}

func NewLoopStartStateImpl() *LoopStartStateImpl {
	baseState := statelang.NewBaseState()
	baseState.SetType(constant.StateTypeLoopStart)
	return &LoopStartStateImpl{
		BaseState: baseState,
	}
}
