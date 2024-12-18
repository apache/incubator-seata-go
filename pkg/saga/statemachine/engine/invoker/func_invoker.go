package invoker

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/util/log"
)

type FuncInvoker struct {
	ServicesMapLock sync.Mutex
	servicesMap     map[string]FuncService
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
	methodLock  sync.Mutex
	method      any
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

	retryCountMap := make(map[state.Retry]int)
	for {
		res, err, shouldRetry := func() (res []reflect.Value, resErr error, shouldRetry bool) {
			defer func() {
				if r := recover(); r != nil {
					errStr := fmt.Sprintf("%v", r)
					retry := f.matchRetry(serviceTaskStateImpl, errStr)
					res = nil
					resErr = errors.New(errStr)

					if retry == nil {
						return
					}
					shouldRetry = f.needRetry(serviceTaskStateImpl, retryCountMap, retry, resErr)
					return
				}
			}()

			outs := method.Call(args)

			if err, ok := outs[len(outs)-1].Interface().(error); ok {
				errStr := err.Error()
				retry := f.matchRetry(serviceTaskStateImpl, errStr)
				res = nil
				resErr = err

				if retry == nil {
					return
				}

				shouldRetry = f.needRetry(serviceTaskStateImpl, retryCountMap, retry, resErr)
				return
			}

			res = outs
			resErr = nil
			shouldRetry = false
			return
		}()

		if !shouldRetry {
			if err != nil {
				return nil, errors.New("invoke service[" + serviceTaskStateImpl.ServiceName() + "]." + serviceTaskStateImpl.ServiceMethod() + " failed, err is " + err.Error())
			}
			return res, nil
		}
	}
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

func (f *FuncServiceImpl) matchRetry(impl *state.ServiceTaskStateImpl, str string) state.Retry {
	if impl.Retry() != nil {
		for _, retry := range impl.Retry() {
			if retry.Exceptions() != nil {
				for _, exception := range retry.Exceptions() {
					if strings.Contains(str, exception) {
						return retry
					}
				}
			}
		}
	}
	return nil
}

func (f *FuncServiceImpl) needRetry(impl *state.ServiceTaskStateImpl, countMap map[state.Retry]int, retry state.Retry, err error) bool {
	attempt, exist := countMap[retry]
	if !exist {
		countMap[retry] = 0
	}

	if attempt >= retry.MaxAttempt() {
		return false
	}

	intervalSecond := retry.IntervalSecond()
	backoffRate := retry.BackoffRate()
	var currentInterval int64
	if attempt == 0 {
		currentInterval = int64(intervalSecond * 1000)
	} else {
		currentInterval = int64(intervalSecond * backoffRate * float64(attempt) * 1000)
	}

	log.Warnf("invoke service[%s.%s] failed, will retry after %s millis, current retry count: %s, current err: %s",
		impl.ServiceName(), impl.ServiceMethod(), currentInterval, attempt, err)

	time.Sleep(time.Duration(currentInterval) * time.Millisecond)
	countMap[retry] = attempt + 1
	return true
}
