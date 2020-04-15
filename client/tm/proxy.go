package proxy

import (
	"context"
	context2 "github.com/dk-lockdown/seata-golang/client/context"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"reflect"
)

type GlobalTransactionProxyService interface {
	GetProxiedService() GlobalTransactionService
	GetMethodTransactionInfo(methodName string) TransactionInfo
}

var (
	typError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()
)

func Implement(v GlobalTransactionProxyService) {
	// check parameters, incoming interface must be a elem's pointer.
	valueOf := reflect.ValueOf(v)
	logging.Logger.Debugf("[Implement] reflect.TypeOf: %s", valueOf.String())

	valueOfElem := valueOf.Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		logging.Logger.Errorf("%s must be a struct ptr", valueOf.String())
		return
	}
	proxiedService := v.GetProxiedService()
	pxdService := reflect.ValueOf(proxiedService)
	serviceName := reflect.Indirect(pxdService).Type().Name()

	makeCallProxy := func(serviceName, methodName string,info TransactionInfo) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			var (
				args         = make([]interface{},0)
				returnValues = make([]reflect.Value,0)
			)

			inNum := len(in)
			invCtx := &context2.RootContext{Context: context.Background()}
			for i := 0; i < inNum; i++ {
				if in[i].Type().String() == "context.Context" {
					if !in[i].IsNil() {
						// the user declared context as method's parameter
						invCtx =  &context2.RootContext{Context:in[i].Interface().(context.Context)}
					}
				}
				args = append(args,in[i].Interface())
			}

			returnValues,_ = Invoke(invCtx,serviceName,methodName,args)

			return returnValues
		}
	}

	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		methodName := t.Name
		f := valueOfElem.Field(i)
		if f.Kind() == reflect.Func && f.IsValid() && f.CanSet() {
			outNum := t.Type.NumOut()

			// The latest return type of the method must be error.
			if returnType := t.Type.Out(outNum - 1); returnType != typError {
				logging.Logger.Warnf("the latest return type %s of method %q is not error", returnType, t.Name)
				continue
			}

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeCallProxy(serviceName,methodName,v.GetMethodTransactionInfo(methodName))))
			logging.Logger.Debugf("set method [%s]", methodName)
		}
	}
}