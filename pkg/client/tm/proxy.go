package tm

import (
	"context"
	"reflect"

	"github.com/pkg/errors"

	ctx "github.com/opentrx/seata-golang/v2/pkg/client/base/context"
	"github.com/opentrx/seata-golang/v2/pkg/client/base/model"
	"github.com/opentrx/seata-golang/v2/pkg/client/proxy"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
)

type GlobalTransactionProxyService interface {
	GetProxyService() interface{}
	GetMethodTransactionInfo(methodName string) *model.TransactionInfo
}

var (
	typError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()
)

func Implement(v GlobalTransactionProxyService) {
	valueOf := reflect.ValueOf(v)
	log.Debugf("[implement] reflect.TypeOf: %s", valueOf.String())

	valueOfElem := valueOf.Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		log.Errorf("%s must be a struct ptr", valueOf.String())
		return
	}
	proxyService := v.GetProxyService()

	makeCallProxy := func(methodDesc *proxy.MethodDescriptor, txInfo *model.TransactionInfo) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			var (
				args                     []interface{}
				returnValues             []reflect.Value
				suspendedResourcesHolder *SuspendedResourcesHolder
			)

			if txInfo == nil {
				// testing phase, this problem should be resolved
				panic(errors.New("transactionInfo does not exist"))
			}

			inNum := len(in)
			if inNum+1 != methodDesc.ArgsNum {
				// testing phase, this problem should be resolved
				panic(errors.New("args does not match"))
			}

			invCtx := ctx.NewRootContext(context.Background())
			for i := 0; i < inNum; i++ {
				if in[i].Type().String() == "context.Context" {
					if !in[i].IsNil() {
						// the user declared context as method's parameter
						invCtx = ctx.NewRootContext(in[i].Interface().(context.Context))
					}
				}
				args = append(args, in[i].Interface())
			}

			tx := GetCurrentOrCreate(invCtx)
			defer func() {
				err := tx.Resume(suspendedResourcesHolder, invCtx)
				if err != nil {
					log.Error(err)
				}
			}()

			switch txInfo.Propagation {
			case model.Required:
			case model.RequiresNew:
				suspendedResourcesHolder, _ = tx.Suspend(true, invCtx)
			case model.NotSupported:
				suspendedResourcesHolder, _ = tx.Suspend(true, invCtx)
				returnValues = proxy.Invoke(methodDesc, invCtx, args)
				return returnValues
			case model.Supports:
				if !invCtx.InGlobalTransaction() {
					returnValues = proxy.Invoke(methodDesc, invCtx, args)
					return returnValues
				}
			case model.Never:
				if invCtx.InGlobalTransaction() {
					return proxy.ReturnWithError(methodDesc, errors.Errorf("Existing transaction found for transaction marked with propagation 'never',xid = %s", invCtx.GetXID()))
				}
				returnValues = proxy.Invoke(methodDesc, invCtx, args)
				return returnValues
			case model.Mandatory:
				if !invCtx.InGlobalTransaction() {
					return proxy.ReturnWithError(methodDesc, errors.New("No existing transaction found for transaction marked with propagation 'mandatory'"))
				}
			default:
				return proxy.ReturnWithError(methodDesc, errors.Errorf("Not Supported Propagation: %s", txInfo.Propagation.String()))
			}

			beginErr := tx.BeginWithTimeoutAndName(txInfo.TimeOut, txInfo.Name, invCtx)
			if beginErr != nil {
				return proxy.ReturnWithError(methodDesc, errors.WithStack(beginErr))
			}

			returnValues = proxy.Invoke(methodDesc, invCtx, args)

			errValue := returnValues[len(returnValues)-1]

			// todo 只要出错就回滚，未来可以优化一下，某些错误才回滚，某些错误的情况下，可以提交
			if errValue.IsValid() && !errValue.IsNil() {
				rollbackErr := tx.Rollback(invCtx)
				if rollbackErr != nil {
					return proxy.ReturnWithError(methodDesc, errors.WithStack(rollbackErr))
				}
				return returnValues
			}

			commitErr := tx.Commit(invCtx)
			if commitErr != nil {
				return proxy.ReturnWithError(methodDesc, errors.WithStack(commitErr))
			}

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
				log.Warnf("the latest return type %s of method %q is not error", returnType, t.Name)
				continue
			}

			methodDescriptor := proxy.Register(proxyService, methodName)

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeCallProxy(methodDescriptor, v.GetMethodTransactionInfo(methodName))))
			log.Debugf("set method [%s]", methodName)
		}
	}
}
