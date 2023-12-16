package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineEngine interface {
	//TODO check
	start(ctx context.Context, args ...interface{}) (statelang.StateMachineInstance, error)
}
