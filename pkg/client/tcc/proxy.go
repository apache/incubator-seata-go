package tcc

import (
	"encoding/json"
	"reflect"

	gxnet "github.com/dubbogo/gost/net"
	"github.com/pkg/errors"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	ctx "github.com/opentrx/seata-golang/v2/pkg/client/base/context"
	"github.com/opentrx/seata-golang/v2/pkg/client/proxy"
	"github.com/opentrx/seata-golang/v2/pkg/client/rm"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	"github.com/opentrx/seata-golang/v2/pkg/util/time"
)

var (
	TccActionName = "TccActionName"

	TryMethod     = "Try"
	ConfirmMethod = "Confirm"
	CancelMethod  = "Cancel"

	ActionStartTime = "action-start-time"
	ActionName      = "actionName"
	PrepareMethod   = "sys::prepare"
	CommitMethod    = "sys::commit"
	RollbackMethod  = "sys::rollback"
	HostName        = "host-name"

	TccMethodArguments = "arguments"
	TccMethodResult    = "result"

	businessActionContextType = reflect.TypeOf(&ctx.BusinessActionContext{})
)

type TccService interface {
	Try(ctx *ctx.BusinessActionContext, async bool) (bool, error)
	Confirm(ctx *ctx.BusinessActionContext) bool
	Cancel(ctx *ctx.BusinessActionContext) bool
}

type TccProxyService interface {
	GetTccService() TccService
}

func ImplementTCC(v TccProxyService) {
	valueOf := reflect.ValueOf(v)
	log.Debugf("[implement] reflect.TypeOf: %s", valueOf.String())

	valueOfElem := valueOf.Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		log.Errorf("%s must be a struct ptr", valueOf.String())
		return
	}
	proxyService := v.GetTccService()
	makeCallProxy := func(methodDesc *proxy.MethodDescriptor, resource *TCCResource) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			businessContextValue := in[0]
			async := in[1].Interface().(bool)
			businessActionContext := businessContextValue.Interface().(*ctx.BusinessActionContext)
			rootContext := businessActionContext.RootContext
			businessActionContext.XID = rootContext.GetXID()
			businessActionContext.ActionName = resource.ActionName
			if !rootContext.InGlobalTransaction() {
				args := make([]interface{}, 0)
				args = append(args, businessActionContext)
				args = append(args, async)
				return proxy.Invoke(methodDesc, nil, args)
			}

			returnValues, err := proceed(methodDesc, businessActionContext, async, resource)
			if err != nil {
				return proxy.ReturnWithError(methodDesc, errors.WithStack(err))
			}
			return returnValues
		}
	}

	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		methodName := t.Name
		f := valueOfElem.Field(i)
		if f.Kind() == reflect.Func && f.IsValid() && f.CanSet() && methodName == TryMethod {
			if t.Type.NumIn() != 1 && t.Type.In(0) != businessActionContextType {
				panic("prepare method argument is not BusinessActionContext")
			}

			actionName := t.Tag.Get(TccActionName)
			if actionName == "" {
				panic("must tag TccActionName")
			}

			commitMethodDesc := proxy.Register(proxyService, ConfirmMethod)
			cancelMethodDesc := proxy.Register(proxyService, CancelMethod)
			tryMethodDesc := proxy.Register(proxyService, methodName)

			tccResource := &TCCResource{
				ActionName:         actionName,
				PrepareMethodName:  TryMethod,
				CommitMethodName:   ConfirmMethod,
				CommitMethod:       commitMethodDesc,
				RollbackMethodName: CancelMethod,
				RollbackMethod:     cancelMethodDesc,
			}

			tccResourceManager.RegisterResource(tccResource)

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeCallProxy(tryMethodDesc, tccResource)))
			log.Debugf("set method [%s]", methodName)
		}
	}
}

func proceed(methodDesc *proxy.MethodDescriptor, ctx *ctx.BusinessActionContext, async bool, resource *TCCResource) ([]reflect.Value, error) {
	var (
		args = make([]interface{}, 0)
	)

	branchID, err := doTccActionLogStore(ctx, resource)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ctx.BranchID = branchID

	args = append(args, ctx)
	args = append(args, async)
	returnValues := proxy.Invoke(methodDesc, nil, args)
	errValue := returnValues[len(returnValues)-1]
	if errValue.IsValid() && !errValue.IsNil() {
		err := rm.GetResourceManager().BranchReport(ctx.RootContext, ctx.XID, branchID, apis.TCC, apis.PhaseOneFailed, nil)
		if err != nil {
			log.Errorf("branch report err: %v", err)
		}
	}

	return returnValues, nil
}

func doTccActionLogStore(ctx *ctx.BusinessActionContext, resource *TCCResource) (int64, error) {
	ctx.ActionContext[ActionStartTime] = time.CurrentTimeMillis()
	ctx.ActionContext[PrepareMethod] = resource.PrepareMethodName
	ctx.ActionContext[CommitMethod] = resource.CommitMethodName
	ctx.ActionContext[RollbackMethod] = resource.RollbackMethodName
	ctx.ActionContext[ActionName] = ctx.ActionName
	ip, err := gxnet.GetLocalIP()
	if err == nil {
		ctx.ActionContext[HostName] = ip
	} else {
		log.Warn("getLocalIP error")
	}

	applicationContext := make(map[string]interface{})
	applicationContext[TccActionContext] = ctx.ActionContext

	applicationData, err := json.Marshal(applicationContext)
	if err != nil {
		log.Errorf("marshal applicationContext failed:%v", applicationContext)
		return 0, err
	}

	branchID, err := rm.GetResourceManager().BranchRegister(
		ctx.RootContext,
		ctx.XID,
		resource.GetResourceID(),
		resource.GetBranchType(),
		applicationData,
		"",
		ctx.AsyncCommit,
	)
	if err != nil {
		log.Errorf("TCC branch Register error, xid: %s", ctx.XID)
		return 0, errors.WithStack(err)
	}
	return branchID, nil
}
