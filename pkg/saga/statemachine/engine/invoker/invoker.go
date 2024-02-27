package invoker

import (
	"context"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang/state"
	"reflect"
	"sync"
)

type ScriptInvokerManager interface {
}

type ScriptInvoker interface {
}

type ServiceInvokerManager interface {
	ServiceInvoker(serviceType string) ServiceInvoker
	PutServiceInvoker(serviceType string, invoker ServiceInvoker)
}

type ServiceInvoker interface {
	Invoke(ctx context.Context, input []any, service state.ServiceTaskState) (output []reflect.Value, err error)
	Close(ctx context.Context) error
}

type ServiceInvokerManagerImpl struct {
	invokers map[string]ServiceInvoker
	mutex    sync.Mutex
}

func NewServiceInvokerManagerImpl() *ServiceInvokerManagerImpl {
	return &ServiceInvokerManagerImpl{
		invokers: make(map[string]ServiceInvoker),
	}
}

func (manager *ServiceInvokerManagerImpl) ServiceInvoker(serviceType string) ServiceInvoker {
	return manager.invokers[serviceType]
}

func (manager *ServiceInvokerManagerImpl) PutServiceInvoker(serviceType string, invoker ServiceInvoker) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.invokers[serviceType] = invoker
}
