package rpc

import (
	"sync"
	"sync/atomic"
)

var ServiceStatusMap sync.Map

type Status struct {
	Active int32
	Total  int32
}

// RemoveStatus remove the RpcStatus of this service
func RemoveStatus(service string) {
	ServiceStatusMap.Delete(service)
}

// BeginCount begin count
func BeginCount(service string) {
	status := GetStatus(service)
	atomic.AddInt32(&status.Active, 1)
}

// EndCount end count
func EndCount(service string) {
	status := GetStatus(service)
	atomic.AddInt32(&status.Active, -1)
	atomic.AddInt32(&status.Total, 1)
}

// GetStatus get status
func GetStatus(service string) *Status {
	a, _ := ServiceStatusMap.LoadOrStore(service, new(Status))
	return a.(*Status)
}

// GetActive get active.
func (s *Status) GetActive() int32 {
	return s.Active
}

// GetTotal get total.
func (s *Status) GetTotal() int32 {
	return s.Total
}
