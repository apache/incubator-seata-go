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

package saga

import (
	"context"
	"encoding/json"
	"fmt"
	gostnet "github.com/dubbogo/gost/net"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	"reflect"
	"sync"
	"time"
)

type SagaServiceProxy struct {
	referenceName        string
	registerResourceOnce sync.Once
	*SagaResource
}

func NewSagaServiceProxy(service interface{}) (*SagaServiceProxy, error) {
	sagaResource, err := ParseSagaResource(service)
	if err != nil {
		log.Errorf("invalid saga service, err %v", err)
	}
	proxy := &SagaServiceProxy{
		SagaResource: sagaResource,
	}
	return proxy, proxy.RegisterResource()
}

func (proxy *SagaServiceProxy) RegisterResource() error {
	var err error
	proxy.registerResourceOnce.Do(func() {
		err = rm.GetRmCacheInstance().GetResourceManager(branch.BranchTypeSAGA).RegisterResource(proxy.SagaResource)
		if err != nil {
			log.Errorf("NewSagaServiceProxy RegisterResource error: %#v", err.Error())
		}
	})
	return err
}

func (proxy *SagaServiceProxy) initActionContext(params interface{}) map[string]interface{} {
	actionContext := proxy.getActionContextParameters(params)
	actionContext[constant.ActionStartTime] = time.Now().UnixNano() / 1e6
	actionContext[constant.ActionMethod] = proxy.SagaResource.SagaAction.GetNormalActionName()
	actionContext[constant.CompensationMethod] = proxy.SagaResource.SagaAction.GetCompensationName()
	actionContext[constant.ActionName] = proxy.SagaResource.SagaAction.GetActionName(params)
	actionContext[constant.HostName], _ = gostnet.GetLocalIP()
	return actionContext
}

// getOrCreateBusinessActionContext When the parameters of the prepare method are the following scenarios, obtain the context in the following ways：
// 1. null: create new BusinessActionContext
// 2. tm.BusinessActionContext: return it
// 3. *tm.BusinessActionContext: if nil then create new BusinessActionContext, else return it
// 4. Struct: if there is an attribute of businessactioncontext enum and it is not nil, return it
// 5. else: create new BusinessActionContext
func (proxy *SagaServiceProxy) getOrCreateBusinessActionContext(params interface{}) *tm.BusinessActionContext {
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

func (proxy *SagaServiceProxy) registerBranch(ctx context.Context, params interface{}) error {
	if !tm.IsGlobalTx(ctx) {
		errStr := "BranchRegister error, transaction should be opened"
		log.Errorf(errStr)
		return fmt.Errorf(errStr)
	}

	sagaContext := proxy.initBusinessActionContext(ctx, params)
	actionContext := proxy.initActionContext(params)
	for k, v := range actionContext {
		sagaContext.ActionContext[k] = v
	}

	applicationData, _ := json.Marshal(map[string]interface{}{
		constant.ActionContext: actionContext,
	})
	branchId, err := rm.GetRMRemotingInstance().BranchRegister(rm.BranchRegisterParam{
		BranchType:      branch.BranchTypeSAGA,
		ResourceId:      proxy.GetSagaActionName(),
		ClientId:        "",
		Xid:             tm.GetXID(ctx),
		ApplicationData: string(applicationData),
		LockKeys:        "",
	})
	if err != nil {
		log.Errorf("register branch transaction error %s ", err.Error())
		return err
	}
	sagaContext.BranchId = branchId
	tm.SetBusinessActionContext(ctx, sagaContext)
	return nil
}

func (proxy *SagaServiceProxy) getActionContextParameters(params interface{}) map[string]interface{} {
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
		tagVal, hasTag := structField.Tag.Lookup(constant.SagaBusinessActionContextParameter)
		if !hasTag || tagVal == `-` || tagVal == "" {
			continue
		}
		actionContext[tagVal] = val.Field(i).Interface()
	}
	return actionContext
}

// initBusinessActionContext init saga context
func (t *SagaServiceProxy) initBusinessActionContext(ctx context.Context, params interface{}) *tm.BusinessActionContext {
	sagaContext := t.getOrCreateBusinessActionContext(params)
	sagaContext.Xid = tm.GetXID(ctx)
	sagaContext.ActionName = t.GetSagaActionName()
	// todo read from config file
	sagaContext.IsDelayReport = true
	if sagaContext.ActionContext == nil {
		sagaContext.ActionContext = make(map[string]interface{}, 0)
	}
	return sagaContext
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

func (proxy *SagaServiceProxy) GetTransactionInfo() tm.GtxConfig {
	// todo replace with config
	return tm.GtxConfig{
		Timeout: time.Second * 10,
		Name:    proxy.GetSagaActionName(),
		// Propagation, Propagation
		// LockRetryInternal, int64
		// LockRetryTimes    int64
	}
}
