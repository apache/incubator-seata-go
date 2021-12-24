package proxy

import (
	"context"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"

	ctx "github.com/opentrx/seata-golang/v2/pkg/client/base/context"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
)

var (
	// serviceDescriptorMap, string -> *ServiceDescriptor
	serviceDescriptorMap = sync.Map{}
)

// MethodDescriptor
type MethodDescriptor struct {
	Method           reflect.Method
	CallerValue      reflect.Value
	CtxType          reflect.Type
	ArgsType         []reflect.Type
	ArgsNum          int
	ReturnValuesType []reflect.Type
	ReturnValuesNum  int
}

// ServiceDescriptor
type ServiceDescriptor struct {
	Name         string
	ReflectType  reflect.Type
	ReflectValue reflect.Value
	Methods      sync.Map // string -> *MethodDescriptor
}

// Register
func Register(service interface{}, methodName string) *MethodDescriptor {
	serviceType := reflect.TypeOf(service)
	serviceValue := reflect.ValueOf(service)
	svcName := reflect.Indirect(serviceValue).Type().Name()

	svcDesc, _ := serviceDescriptorMap.LoadOrStore(svcName, &ServiceDescriptor{
		Name:         svcName,
		ReflectType:  serviceType,
		ReflectValue: serviceValue,
		Methods:      sync.Map{},
	})
	svcDescriptor := svcDesc.(*ServiceDescriptor)
	methodDesc, methodExist := svcDescriptor.Methods.Load(methodName)
	if methodExist {
		methodDescriptor := methodDesc.(*MethodDescriptor)
		return methodDescriptor
	}

	method, methodFounded := serviceType.MethodByName(methodName)
	if methodFounded {
		methodDescriptor := describeMethod(method)
		if methodDescriptor != nil {
			methodDescriptor.CallerValue = serviceValue
			svcDescriptor.Methods.Store(methodName, methodDescriptor)
			return methodDescriptor
		}
	}
	return nil
}

// describeMethod
// might return nil when method is not exported or some other error
func describeMethod(method reflect.Method) *MethodDescriptor {
	methodType := method.Type
	methodName := method.Name
	inNum := methodType.NumIn()
	outNum := methodType.NumOut()

	// Method must be exported.
	if method.PkgPath != "" {
		return nil
	}

	var (
		ctxType                    reflect.Type
		argsType, returnValuesType []reflect.Type
	)

	for index := 1; index < inNum; index++ {
		if methodType.In(index).String() == "context.Context" {
			ctxType = methodType.In(index)
		}
		argsType = append(argsType, methodType.In(index))
		// need not be a pointer.
		if !isExportedOrBuiltinType(methodType.In(index)) {
			log.Errorf("argument type of method %q is not exported %v", methodName, methodType.In(index))
			return nil
		}
	}

	// returnValuesType
	for num := 0; num < outNum; num++ {
		returnValuesType = append(returnValuesType, methodType.Out(num))
	}

	return &MethodDescriptor{
		Method:           method,
		CtxType:          ctxType,
		ArgsType:         argsType,
		ArgsNum:          inNum,
		ReturnValuesType: returnValuesType,
		ReturnValuesNum:  outNum,
	}
}

// Is this an exported - upper case - name
func isExported(name string) bool {
	s, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(s)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// Invoke
func Invoke(methodDesc *MethodDescriptor, ctx *ctx.RootContext, args []interface{}) []reflect.Value {

	in := []reflect.Value{methodDesc.CallerValue}

	for i := 0; i < len(args); i++ {
		t := reflect.ValueOf(args[i])
		if methodDesc.ArgsType[i].String() == "context.Context" {
			t = SuiteContext(ctx, methodDesc)
		}
		if !t.IsValid() {
			at := methodDesc.ArgsType[i]
			if at.Kind() == reflect.Ptr {
				at = at.Elem()
			}
			t = reflect.New(at)
		}
		in = append(in, t)
	}

	returnValues := methodDesc.Method.Func.Call(in)

	return returnValues
}

func SuiteContext(ctx context.Context, methodDesc *MethodDescriptor) reflect.Value {
	if contextValue := reflect.ValueOf(ctx); contextValue.IsValid() {
		return contextValue
	}
	return reflect.Zero(methodDesc.CtxType)
}

func ReturnWithError(methodDesc *MethodDescriptor, err error) []reflect.Value {
	var result = make([]reflect.Value, 0)
	for i := 0; i < methodDesc.ReturnValuesNum-1; i++ {
		result = append(result, reflect.Zero(methodDesc.ReturnValuesType[i]))
	}
	result = append(result, reflect.ValueOf(err))
	return result
}
