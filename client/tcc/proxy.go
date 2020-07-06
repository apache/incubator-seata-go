package tcc

import (
	"encoding/json"
	"reflect"
	"strconv"
)

import (
	gxnet "github.com/dubbogo/gost/net"
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/client/context"
	"github.com/dk-lockdown/seata-golang/client/proxy"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/pkg/time"
)

var (
	TCC_ACTION_NAME = "TccActionName"

	TRY_METHOD     = "Try"
	CONFIRM_METHOD = "Confirm"
	CANCEL_METHOD  = "Cancel"

	ACTION_START_TIME = "action-start-time"
	ACTION_NAME       = "actionName"
	PREPARE_METHOD    = "sys::prepare"
	COMMIT_METHOD     = "sys::commit"
	ROLLBACK_METHOD   = "sys::rollback"
	HOST_NAME         = "host-name"

	TCC_METHOD_ARGUMENTS = "arguments"
	TCC_METHOD_RESULT    = "result"

	businessActionContextType = reflect.TypeOf(&context.BusinessActionContext{})
)

type TccService interface {
	Try(ctx *context.BusinessActionContext) (bool, error)
	Confirm(ctx *context.BusinessActionContext) bool
	Cancel(ctx *context.BusinessActionContext) bool
}

type TccProxyService interface {
	GetTccService() TccService
}

func ImplementTCC(v TccProxyService) {
	valueOf := reflect.ValueOf(v)
	logging.Logger.Debugf("[Implement] reflect.TypeOf: %s", valueOf.String())

	valueOfElem := valueOf.Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		logging.Logger.Errorf("%s must be a struct ptr", valueOf.String())
		return
	}
	proxyService := v.GetTccService()
	makeCallProxy := func(methodDesc *proxy.MethodDescriptor, resource *TCCResource) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			businessContextValue := in[0]
			businessActionContext := businessContextValue.Interface().(*context.BusinessActionContext)
			rootContext := businessActionContext.RootContext
			businessActionContext.Xid = rootContext.GetXID()
			businessActionContext.ActionName = resource.ActionName
			if !rootContext.InGlobalTransaction() {
				args := make([]interface{}, 0)
				args = append(args, businessActionContext)
				return proxy.Invoke(methodDesc, nil, args)
			}

			returnValues, _ := proceed(methodDesc, businessActionContext, resource)
			return returnValues
		}
	}

	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		methodName := t.Name
		f := valueOfElem.Field(i)
		if f.Kind() == reflect.Func && f.IsValid() && f.CanSet() && methodName == TRY_METHOD {
			if t.Type.NumIn() != 1 && t.Type.In(0) != businessActionContextType {
				panic("prepare method argument is not BusinessActionContext")
			}

			actionName := t.Tag.Get(TCC_ACTION_NAME)
			if actionName == "" {
				panic("must tag TccActionName")
			}

			commitMethodDesc := proxy.Register(proxyService, CONFIRM_METHOD)
			cancelMethodDesc := proxy.Register(proxyService, CANCEL_METHOD)
			tryMethodDesc := proxy.Register(proxyService, methodName)

			tccResource := &TCCResource{
				ResourceGroupId:    "",
				AppName:            "",
				ActionName:         actionName,
				PrepareMethodName:  TRY_METHOD,
				CommitMethodName:   COMMIT_METHOD,
				CommitMethod:       commitMethodDesc,
				RollbackMethodName: CANCEL_METHOD,
				RollbackMethod:     cancelMethodDesc,
			}

			tccResourceManager.RegisterResource(tccResource)

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeCallProxy(tryMethodDesc, tccResource)))
			logging.Logger.Debugf("set method [%s]", methodName)
		}
	}
}

func proceed(methodDesc *proxy.MethodDescriptor, ctx *context.BusinessActionContext, resource *TCCResource) ([]reflect.Value, error) {
	var (
		args = make([]interface{}, 0)
	)

	branchId, err := doTccActionLogStore(ctx, resource)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ctx.BranchId = branchId

	args = append(args, ctx)
	returnValues := proxy.Invoke(methodDesc, nil, args)

	return returnValues, nil
}

func doTccActionLogStore(ctx *context.BusinessActionContext, resource *TCCResource) (string, error) {
	ctx.ActionContext[ACTION_START_TIME] = time.CurrentTimeMillis()
	ctx.ActionContext[PREPARE_METHOD] = resource.PrepareMethodName
	ctx.ActionContext[COMMIT_METHOD] = resource.CommitMethodName
	ctx.ActionContext[ROLLBACK_METHOD] = resource.RollbackMethodName
	ctx.ActionContext[ACTION_NAME] = ctx.ActionName
	ip, err := gxnet.GetLocalIP()
	if err == nil {
		ctx.ActionContext[HOST_NAME] = ip
	} else {
		logging.Logger.Warn("getLocalIP error")
	}

	applicationContext := make(map[string]interface{})
	applicationContext[TCC_ACTION_CONTEXT] = ctx.ActionContext

	applicationData, err := json.Marshal(applicationContext)
	if err != nil {
		logging.Logger.Errorf("marshal applicationContext failed:%v", applicationContext)
		return "", err
	}

	branchId, err := tccResourceManager.BranchRegister(meta.BranchTypeTCC, ctx.ActionName, "", ctx.Xid, applicationData, "")
	if err != nil {
		logging.Logger.Errorf("TCC branch Register error, xid: %s", ctx.Xid)
		return "", errors.WithStack(err)
	}
	return strconv.FormatInt(branchId, 10), nil
}
