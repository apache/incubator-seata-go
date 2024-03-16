/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tcc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	gostnet "github.com/dubbogo/gost/net"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
	"seata.apache.org/seata-go/pkg/util/reflectx"
)

type TCCServiceProxy struct {
	referenceName        string
	registerResourceOnce sync.Once
	*TCCResource
}

func NewTCCServiceProxy(service interface{}) (*TCCServiceProxy, error) {
	tccResource, err := ParseTCCResource(service)
	if err != nil {
		log.Errorf("invalid tcc service, err %v", err)
		return nil, err
	}
	proxy := &TCCServiceProxy{
		TCCResource: tccResource,
	}
	return proxy, proxy.RegisterResource()
}

func (t *TCCServiceProxy) RegisterResource() error {
	var err error
	t.registerResourceOnce.Do(func() {
		err = rm.GetRmCacheInstance().GetResourceManager(branch.BranchTypeTCC).RegisterResource(t.TCCResource)
		if err != nil {
			log.Errorf("NewTCCServiceProxy RegisterResource error: %#v", err.Error())
		}
	})
	return err
}

func (t *TCCServiceProxy) SetReferenceName(referenceName string) {
	t.referenceName = referenceName
}

func (t *TCCServiceProxy) Reference() string {
	if t.referenceName != "" {
		return t.referenceName
	}
	return reflectx.GetReference(t.TCCResource.TwoPhaseAction.GetTwoPhaseService())
}

func (t *TCCServiceProxy) Prepare(ctx context.Context, params interface{}) (interface{}, error) {
	if tm.IsGlobalTx(ctx) {
		err := t.registeBranch(ctx, params)
		if err != nil {
			return nil, err
		}
	}

	// to set up the fence phase
	tm.SetFencePhase(ctx, enum.FencePhasePrepare)
	return t.TCCResource.Prepare(ctx, params)
}

// registeBranch send register branch transaction request
func (t *TCCServiceProxy) registeBranch(ctx context.Context, params interface{}) error {
	if !tm.IsGlobalTx(ctx) {
		errStr := "BranchRegister error, transaction should be opened"
		log.Errorf(errStr)
		return fmt.Errorf(errStr)
	}

	tccContext := t.initBusinessActionContext(ctx, params)
	actionContext := t.initActionContext(params)
	for k, v := range actionContext {
		tccContext.ActionContext[k] = v
	}

	applicationData, _ := json.Marshal(map[string]interface{}{
		constant.ActionContext: actionContext,
	})
	branchId, err := rm.GetRMRemotingInstance().BranchRegister(rm.BranchRegisterParam{
		BranchType:      branch.BranchTypeTCC,
		ResourceId:      t.GetActionName(),
		ClientId:        "",
		Xid:             tm.GetXID(ctx),
		ApplicationData: string(applicationData),
		LockKeys:        "",
	})
	if err != nil {
		log.Errorf("register branch transaction error %s ", err.Error())
		return err
	}
	tccContext.BranchId = branchId
	tm.SetBusinessActionContext(ctx, tccContext)
	return nil
}

// initActionContext init action context
func (t *TCCServiceProxy) initActionContext(params interface{}) map[string]interface{} {
	actionContext := t.getActionContextParameters(params)
	actionContext[constant.ActionStartTime] = time.Now().UnixNano() / 1e6
	actionContext[constant.PrepareMethod] = t.TCCResource.TwoPhaseAction.GetPrepareMethodName()
	actionContext[constant.CommitMethod] = t.TCCResource.TwoPhaseAction.GetCommitMethodName()
	actionContext[constant.RollbackMethod] = t.TCCResource.TwoPhaseAction.GetRollbackMethodName()
	actionContext[constant.ActionName] = t.TCCResource.TwoPhaseAction.GetActionName()
	actionContext[constant.HostName], _ = gostnet.GetLocalIP()
	return actionContext
}

func (t *TCCServiceProxy) getActionContextParameters(params interface{}) map[string]interface{} {
	var (
		actionContext = make(map[string]interface{}, 0)
		typ           reflect.Type
		val           reflect.Value
		isStruct      bool
	)
	if params == nil {
		return actionContext
	}
	if isStruct, val, typ = obtainStructValueType(params); !isStruct {
		return actionContext
	}
	for i := 0; i < typ.NumField(); i++ {
		// skip unexported anonymous filed
		if typ.Field(i).PkgPath != "" {
			continue
		}
		structField := typ.Field(i)
		// skip ignored field
		tagVal, hasTag := structField.Tag.Lookup(constant.TccBusinessActionContextParameter)
		if !hasTag || tagVal == `-` || tagVal == "" {
			continue
		}
		actionContext[tagVal] = val.Field(i).Interface()
	}
	return actionContext
}

// initBusinessActionContext init tcc context
func (t *TCCServiceProxy) initBusinessActionContext(ctx context.Context, params interface{}) *tm.BusinessActionContext {
	tccContext := t.getOrCreateBusinessActionContext(params)
	tccContext.Xid = tm.GetXID(ctx)
	tccContext.ActionName = t.GetActionName()
	// todo read from config file
	tccContext.IsDelayReport = true
	if tccContext.ActionContext == nil {
		tccContext.ActionContext = make(map[string]interface{}, 0)
	}
	return tccContext
}

// getOrCreateBusinessActionContext When the parameters of the prepare method are the following scenarios, obtain the context in the following ways：
// 1. null: create new BusinessActionContext
// 2. tm.BusinessActionContext: return it
// 3. *tm.BusinessActionContext: if nil then create new BusinessActionContext, else return it
// 4. Struct: if there is an attribute of businessactioncontext enum and it is not nil, return it
// 5. else: create new BusinessActionContext
func (t *TCCServiceProxy) getOrCreateBusinessActionContext(params interface{}) *tm.BusinessActionContext {
	if params == nil {
		return &tm.BusinessActionContext{}
	}

	switch params.(type) {
	case tm.BusinessActionContext:
		v := params.(tm.BusinessActionContext)
		return &v
	case *tm.BusinessActionContext:
		v := params.(*tm.BusinessActionContext)
		if v != nil {
			return v
		}
		return &tm.BusinessActionContext{}
	default:
		break
	}

	var (
		typ      reflect.Type
		val      reflect.Value
		isStruct bool
	)
	if isStruct, val, typ = obtainStructValueType(params); !isStruct {
		return &tm.BusinessActionContext{}
	}
	n := typ.NumField()
	for i := 0; i < n; i++ {
		sf := typ.Field(i)
		if sf.Type == rm.TypBusinessContextInterface {
			v := val.Field(i).Interface()
			if v != nil {
				return v.(*tm.BusinessActionContext)
			}
		}
		if sf.Type == reflect.TypeOf(tm.BusinessActionContext{}) && val.Field(i).CanInterface() {
			v := val.Field(i).Interface().(tm.BusinessActionContext)
			return &v
		}
	}
	return &tm.BusinessActionContext{}
}

// obtainStructValueType check o is struct or pointer enum
func obtainStructValueType(o interface{}) (bool, reflect.Value, reflect.Type) {
	v := reflect.ValueOf(o)
	t := reflect.TypeOf(o)
	switch v.Kind() {
	case reflect.Struct:
		return true, v, t
	case reflect.Ptr:
		return true, v.Elem(), t.Elem()
	default:
		return false, v, nil
	}
}

func (t *TCCServiceProxy) GetTransactionInfo() tm.GtxConfig {
	// todo replace with config
	return tm.GtxConfig{
		Timeout: time.Second * 10,
		Name:    t.GetActionName(),
		// Propagation, Propagation
		// LockRetryInternal, int64
		// LockRetryTimes    int64
	}
}
