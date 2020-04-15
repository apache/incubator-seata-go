package proxy

import (
	"context"
	context2 "github.com/dk-lockdown/seata-golang/client/context"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	// PROXY_METHOD ...
	PROXY_METHOD = "ProxyMethods"
)

var (
	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()

	// ServiceMap ...
	ServiceMap = &serviceMap{
		serviceMap: make(map[string]*Service),
	}
)

type ProxyService interface {
	// Methods
	ProxyMethods() map[string]bool
}

// MethodType ...
type MethodType struct {
	method    reflect.Method
	ctxType   reflect.Type
	argsType  []reflect.Type
	returnValuesType []reflect.Type
}

// Method ...
func (m *MethodType) Method() reflect.Method {
	return m.method
}

// ArgsType ...
func (m *MethodType) ArgsType() []reflect.Type {
	return m.argsType
}

// ReturnValuesType ...
func (m *MethodType) ReturnValuesType() []reflect.Type {
	return m.returnValuesType
}

// Service ...
type Service struct {
	name     string
	rcvr     reflect.Value
	rcvrType reflect.Type
	methods  map[string]*MethodType
}

// Method ...
func (s *Service) Method() map[string]*MethodType {
	return s.methods
}

// RcvrType ...
func (s *Service) RcvrType() reflect.Type {
	return s.rcvrType
}

// Rcvr ...
func (s *Service) Rcvr() reflect.Value {
	return s.rcvr
}

type serviceMap struct {
	mutex      sync.RWMutex        // protects the serviceMap
	serviceMap map[string]*Service // service name -> service
}

func (sm *serviceMap) GetService(name string) *Service {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	if s, ok := sm.serviceMap[name]; ok {
		return s
	}
	return nil
}

func (sm *serviceMap) Register(rcvr ProxyService) (string, error) {
	s := new(Service)
	s.rcvrType = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		s := "no service name for type " + s.rcvrType.String()
		logging.Logger.Errorf(s)
		return "", errors.New(s)
	}
	if !isExported(sname) {
		s := "type " + sname + " is not exported"
		logging.Logger.Errorf(s)
		return "", errors.New(s)
	}

	if server := sm.GetService(sname); server != nil {
		return "", errors.New("service already defined: " + sname)
	}
	s.name = sname
	s.methods = make(map[string]*MethodType)

	// Install the methods
	methods := ""
	methods, s.methods = suitableMethods(s.rcvrType)

	if len(s.methods) == 0 {
		s := "type " + sname + " has no exported methods of suitable type"
		logging.Logger.Errorf(s)
		return "", errors.New(s)
	}
	sm.mutex.Lock()
	sm.serviceMap[s.name] = s
	sm.mutex.Unlock()

	return strings.TrimSuffix(methods, ","), nil
}

func (sm *serviceMap) UnRegister(serviceId string) error {
	sm.mutex.RLock()
	_, ok := sm.serviceMap[serviceId]
	if !ok {
		sm.mutex.RUnlock()
		return errors.New("no service for " + serviceId)
	}
	sm.mutex.RUnlock()

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.serviceMap, serviceId)
	return nil
}

func suitableMethods(typ reflect.Type) (string, map[string]*MethodType) {
	methods := make(map[string]*MethodType)
	var mts []string
	logging.Logger.Debugf("[%s] NumMethod is %d", typ.String(), typ.NumMethod())
	method, ok := typ.MethodByName(PROXY_METHOD)
	var transactionMethods map[string]bool
	if ok && method.Type.NumIn() == 1 && method.Type.NumOut() == 1 && method.Type.Out(0).String() == "map[string]bool" {
		transactionMethods = method.Func.Call([]reflect.Value{reflect.New(typ.Elem())})[0].Interface().(map[string]bool)
	}

	for m := 0; m < typ.NumMethod(); m++ {
		method = typ.Method(m)
		_, ok := transactionMethods[method.Name]
		if ok {
			if mt := suiteMethod(method); mt != nil {
				methods[method.Name] = mt
			}
			mts = append(mts, method.Name)
		}

	}
	return strings.Join(mts, ","), methods
}

func suiteMethod(method reflect.Method) *MethodType {
	mtype := method.Type
	mname := method.Name
	inNum := mtype.NumIn()
	outNum := mtype.NumOut()

	// Method must be exported.
	if method.PkgPath != "" {
		return nil
	}

	var (
		ctxType reflect.Type
		argsType, returnValuesType []reflect.Type
	)

	for index := 1; index < inNum; index++ {
		if mtype.In(index).String() == "context.Context" {
			ctxType = mtype.In(index)
		}
		argsType = append(argsType, mtype.In(index))
		// need not be a pointer.
		if !isExportedOrBuiltinType(mtype.In(index)) {
			logging.Logger.Errorf("argument type of method %q is not exported %v", mname, mtype.In(index))
			return nil
		}
	}


	// The latest return type of the method must be error.
	if returnType := mtype.Out(outNum - 1); returnType != typeOfError {
		logging.Logger.Warnf("the latest return type %s of method %q is not error", returnType, mname)
		return nil
	}

	// returnValuesType
	for num := 0; num < outNum; num++ {
		returnValuesType = append(returnValuesType, mtype.Out(num))
		// need not be a pointer.
		if !isExportedOrBuiltinType(mtype.Out(num)) {
			logging.Logger.Errorf("reply type of method %s not exported{%v}", mname, mtype.Out(num))
			return nil
		}
	}

	return &MethodType{method: method, argsType: argsType, returnValuesType: returnValuesType, ctxType: ctxType}
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

func Invoke(ctx *context2.RootContext,serviceName string,methodName string,args []interface{}) []reflect.Value {
	svc := ServiceMap.GetService(serviceName)
	if svc == nil {
		logging.Logger.Errorf("cannot find service [%s]", serviceName)
		panic(errors.Errorf("cannot find service [%s]", serviceName))
	}

	// get method
	method := svc.Method()[methodName]
	if method == nil {
		logging.Logger.Errorf("cannot find method [%s] of service [%s]", methodName, serviceName)
		panic(errors.Errorf("cannot find method [%s] of service [%s]", methodName, serviceName))
	}

	in := []reflect.Value{svc.Rcvr()}

	for i := 0; i < len(args); i++ {
		t := reflect.ValueOf(args[i])
		if method.ArgsType()[i].String() == "context.Context" {
			t = SuiteContext(method,ctx.Context)
		}
		if !t.IsValid() {
			at := method.ArgsType()[i]
			if at.Kind() == reflect.Ptr {
				at = at.Elem()
			}
			t = reflect.New(at)
		}
		in = append(in, t)
	}

	returnValues := method.Method().Func.Call(in)

	return returnValues
}

func SuiteContext(m *MethodType,ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ctxType)
}