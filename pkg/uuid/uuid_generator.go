package uuid

import (
	"github.com/dk-lockdown/seata-golang/pkg/time"
	"sync/atomic"
)

var (
	UUID          int64 = 1000
	serverNodeId        = 1
	UUID_INTERNAL int64 = 2000000000
	initUUID      int64 = 0
)

func GeneratorUUID() int64 {
	id := atomic.AddInt64(&UUID, 1)
	if id >= GetMaxUUID() {
		if UUID >= id {
			newId := id - UUID_INTERNAL
			atomic.CompareAndSwapInt64(&UUID, id, newId)
			return newId
		}
	}
	return id
}

func SetUUID(expect int64, update int64) bool {
	return atomic.CompareAndSwapInt64(&UUID, expect, update)
}

func GetMaxUUID() int64 {
	return UUID_INTERNAL * (int64(serverNodeId) + 1)
}

func GetInitUUID() int64 {
	return initUUID
}

func Init(svrNodeId int) {
	// 2019-01-01 与 java 版 seata 一致
	var base uint64 = 1546272000000
	serverNodeId = svrNodeId
	atomic.CompareAndSwapInt64(&UUID, UUID, UUID_INTERNAL*int64(serverNodeId))
	current := time.CurrentTimeMillis()
	id := atomic.AddInt64(&UUID, int64((current-base)/time.UnixTimeUnitOffset))
	initUUID = id
}
