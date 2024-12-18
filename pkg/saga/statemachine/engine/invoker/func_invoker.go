package invoker

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/util/log"
)

type FuncInvoker struct {
	servicesMap     map[string]FuncService
	ServicesMapLock sync.Mutex
}

func NewFuncInvoker() *FuncInvoker {
	return &FuncInvoker{
		servicesMap: make(map[string]FuncService),
	}
}

func (f *FuncInvoker) RegisterService(serviceName string, service FuncService) {
	f.ServicesMapLock.Lock()
	defer f.ServicesMapLock.Unlock()
	f.servicesMap[serviceName] = service
}

func (f *FuncInvoker) GetService(serviceName string) FuncService {
	f.ServicesMapLock.Lock()
	defer f.ServicesMapLock.Unlock()
	return f.servicesMap[serviceName]
}

func (f *FuncInvoker) Invoke(ctx context.Context, input []any, service state.ServiceTaskState) (output []reflect.Value, err error) {
	serviceTaskStateImpl := service.(*state.ServiceTaskStateImpl)
	FuncService := f.GetService(serviceTaskStateImpl.ServiceName())
	if FuncService == nil {
		return nil, errors.New("no func service " + serviceTaskStateImpl.ServiceName() + " for service task state")
	}

	if serviceTaskStateImpl.IsAsync() {
		go func() {
			_, err := FuncService.CallMethod(serviceTaskStateImpl, input)
			if err != nil {
				log.Errorf("invoke Service[%s].%s failed, err is %s", serviceTaskStateImpl.ServiceName(), serviceTaskStateImpl.ServiceMethod(), err.Error())
			}
		}()
		return nil, nil
	} else {
		return FuncService.CallMethod(serviceTaskStateImpl, input)
	}
}

func (f *FuncInvoker) Close(ctx context.Context) error {
	return nil
}

type FuncService interface {
	CallMethod(ServiceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error)
}

type FuncServiceImpl struct {
	serviceName string
	method      any
	methodLock  sync.Mutex
}

func NewFuncService(serviceName string, method any) *FuncServiceImpl {
	return &FuncServiceImpl{
		serviceName: serviceName,
		method:      method,
	}
}

func (f *FuncServiceImpl) CallMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error) {
	if serviceTaskStateImpl.Method() == nil {
		err := f.initMethod(serviceTaskStateImpl)
		if err != nil {
			return nil, err
		}
	}
	method := serviceTaskStateImpl.Method()

	args := make([]reflect.Value, 0, len(input))
	for _, arg := range input {
		args = append(args, reflect.ValueOf(arg))
	}

	return method.Call(args), nil
}

func (f *FuncServiceImpl) initMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl) error {
	methodName := serviceTaskStateImpl.ServiceMethod()
	f.methodLock.Lock()
	defer f.methodLock.Unlock()
	methodValue := reflect.ValueOf(f.method)
	if methodValue.IsZero() {
		return errors.New("invalid method when func call, serviceName: " + f.serviceName)
	}

	if methodValue.Kind() == reflect.Func {
		serviceTaskStateImpl.SetMethod(&methodValue)
		return nil
	}

	method := methodValue.MethodByName(methodName)
	if method.IsZero() {
		return errors.New("invalid method name when func call, serviceName: " + f.serviceName + ", methodName: " + methodName)
	}
	serviceTaskStateImpl.SetMethod(&method)
	return nil
}
