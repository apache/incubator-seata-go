package proxy

import (
	"context"
	"reflect"
)

type GlobalTransactionalService interface {
	// Method -> Propagation
	TransactionMethod() map[string]string
}

// MethodType ...
type MethodType struct {
	method    reflect.Method
	ctxType   reflect.Type
	argsType  []reflect.Type
	replyType []reflect.Type
}

// Method ...
func (m *MethodType) Method() reflect.Method {
	return m.method
}

// ArgsType ...
func (m *MethodType) ArgsType() []reflect.Type {
	return m.argsType
}

// ReplyType ...
func (m *MethodType) ReplyType() []reflect.Type {
	return m.replyType
}

// SuiteContext ...
func (m *MethodType) SuiteContext(ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ctxType)
}