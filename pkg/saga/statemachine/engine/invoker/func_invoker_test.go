package invoker

import (
	"context"
	"fmt"
	"testing"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type MockFuncClientImpl struct {
	FuncServiceImpl
}

// struct's method test
type mockFuncImpl struct {
}

func (m *mockFuncImpl) SayHelloRight(word string) string {
	fmt.Println("invoke right")
	return word
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
}

func newFuncServiceStructInvoker() ServiceInvoker {
	mockFuncInvoker := NewFuncInvoker()
	mockFuncService := &mockFuncImpl{}
	mockService := NewFuncService("hello", mockFuncService)
	mockFuncInvoker.RegisterService("hello", mockService)
	return mockFuncInvoker
}

// method test
func SayHelloRight(word string) string {
	return word
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
