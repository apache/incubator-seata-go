package state

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type CompensationTriggerState interface {
	statelang.State
}

type CompensationTriggerStateImpl struct {
	*statelang.BaseState
}

func NewCompensationTriggerStateImpl() *CompensationTriggerStateImpl {
	s := &CompensationTriggerStateImpl{
		BaseState: statelang.NewBaseState(),
	}
	s.SetType(constant.StateTypeCompensationTrigger)
	return s
}
