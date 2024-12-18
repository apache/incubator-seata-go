package invoker

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type MockFuncClientImpl struct {
	FuncServiceImpl
}

// struct's method test
type mockFuncImpl struct {
	invokeCount int
}

func (m *mockFuncImpl) SayHelloRight(word string) (string, error) {
	m.invokeCount++
	fmt.Println("invoke right")
	return word, nil
}

func (m *mockFuncImpl) SayHelloRightLater(word string, delay int) (string, error) {
	m.invokeCount++
	if delay == m.invokeCount {
		fmt.Println("invoke right")
		return word, nil
	}
	fmt.Println("invoke fail")
	return "", errors.New("invoke failed")
}

func TestFuncInvokerInvokeStructSucceed(t *testing.T) {
	ctx := context.Background()
	invoker := newFuncServiceStructInvoker()
	values, err := invoker.Invoke(ctx, []any{"hello"}, newFuncHelloServiceTaskState())
	if err != nil {
		t.Error(err)
		return
	}
	if values == nil || len(values) == 0 {
		t.Error("no value in values")
		return
	}
	if values[0].Interface().(string) != "hello" {
		t.Errorf("expect hello, but got %s", values[0].Interface())
	}
	if _, ok := values[1].Interface().(error); ok {
		t.Errorf("expect nil, but got %s", values[1].Interface())
	}
}

func TestFuncInvokerInvokeStructSucceedInRetry(t *testing.T) {
	ctx := context.Background()
	invoker := newFuncServiceStructInvoker()
	values, err := invoker.Invoke(ctx, []any{"hello", 2}, newFuncHelloServiceTaskStateWithRetry())
	if err != nil {
		t.Error(err)
		return
	}
	if values == nil || len(values) == 0 {
		t.Error("no value in values")
		return
	}
	if values[0].Interface().(string) != "hello" {
		t.Errorf("expect hello, but got %s", values[0].Interface())
	}
	if _, ok := values[1].Interface().(error); ok {
		t.Errorf("expect nil, but got %s", values[1].Interface())
	}
}

func TestFuncInvokerInvokeStructFailedInRetry(t *testing.T) {
	ctx := context.Background()
	invoker := newFuncServiceStructInvoker()
	values, err := invoker.Invoke(ctx, []any{"hello", 5}, newFuncHelloServiceTaskStateWithRetry())
	if err == nil {
		t.Error("expect error, but got nil")
		return
	}
	if values != nil {
		t.Errorf("expect nil, but got %v", values)
	}
}

func newFuncServiceStructInvoker() ServiceInvoker {
	mockFuncInvoker := NewFuncInvoker()
	mockFuncService := &mockFuncImpl{}
	mockService := NewFuncService("hello", mockFuncService)
	mockFuncInvoker.RegisterService("hello", mockService)
	return mockFuncInvoker
}

// method test
func SayHelloRight(word string) (string, error) {
	return word, nil
}

func TestFuncInvokerInvokeMethodSucceed(t *testing.T) {
	ctx := context.Background()
	invoker := newFuncServiceMethodInvoker()
	values, err := invoker.Invoke(ctx, []any{"hello"}, newFuncHelloServiceTaskState())
	if err != nil {
		t.Error(err)
		return
	}
	if values == nil || len(values) == 0 {
		t.Error("no value in values")
		return
	}
	if values[0].Interface().(string) != "hello" {
		t.Errorf("expect hello, but got %s", values[0].Interface())
	}
	if _, ok := values[1].Interface().(error); ok {
		t.Errorf("expect nil, but got %s", values[1].Interface())
	}
}

func newFuncServiceMethodInvoker() ServiceInvoker {
	mockFuncInvoker := NewFuncInvoker()
	mockFuncMethod := SayHelloRight
	mockService := NewFuncService("hello", mockFuncMethod)
	mockFuncInvoker.RegisterService("hello", mockService)
	return mockFuncInvoker
}

func newFuncHelloServiceTaskState() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("hello")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("hello")
	serviceTaskStateImpl.SetServiceType("func")
	serviceTaskStateImpl.SetServiceMethod("SayHelloRight")
	return serviceTaskStateImpl
}

func newFuncHelloServiceTaskStateWithRetry() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("hello")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("hello")
	serviceTaskStateImpl.SetServiceType("func")
	serviceTaskStateImpl.SetServiceMethod("SayHelloRightLater")

	retryImpl := &state.RetryImpl{}
	retryImpl.SetExceptions([]string{"fail"})
	retryImpl.SetIntervalSecond(1)
	retryImpl.SetMaxAttempt(3)
	retryImpl.SetBackoffRate(0.9)
	serviceTaskStateImpl.SetRetry([]state.Retry{retryImpl})
	return serviceTaskStateImpl
}
