package state

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type TaskState interface {
	statelang.State

	GetCompensateState() string

	GetStatus() map[string]string

	GetRetry() []Retry

	//golang not need catch, use error?
	//todo support loop
}

type Retry interface {
	GetErrorTypeNames() []string

	GetIntervalSecond() float64

	GetMaxAttempt() int

	GetBackoffRate() float64
}
