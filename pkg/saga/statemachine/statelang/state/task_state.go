package state

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type TaskState interface {
	statelang.State

	CompensateState() string

	Status() map[string]string

	Retry() []Retry
}

type Retry interface {
	ErrorTypeNames() []string

	IntervalSecond() float64

	MaxAttempt() int

	BackoffRate() float64
}
