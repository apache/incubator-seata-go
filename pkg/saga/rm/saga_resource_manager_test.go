package rm

import (
	"sync"
	"testing"
)

func TestGetSagaResourceManager_Singleton(t *testing.T) {
	var wg sync.WaitGroup
	instances := make([]*SagaResourceManager, 10)

	// 并发获取实例，确保单例
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			instances[idx] = GetSagaResourceManager()
		}(i)
	}
	wg.Wait()

	first := instances[0]
	for i, inst := range instances {
		if inst != first {
			t.Errorf("Instance at index %d is not the same as the first instance", i)
		}
	}
}
