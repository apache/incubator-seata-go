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

package rm

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/exception"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	seataErrors "github.com/seata/seata-go/pkg/util/errors"
)

var (
	sagaResourceManagerInstance *SagaResourceManager
	once                        sync.Once
)

type SagaResourceManager struct {
	rmRemoting    *rm.RMRemoting
	resourceCache sync.Map
}

func InitSaga() {
	rm.GetRmCacheInstance().RegisterResourceManager(GetSagaResourceManager())
}

func GetSagaResourceManager() *SagaResourceManager {
	once.Do(func() {
		sagaResourceManagerInstance = &SagaResourceManager{
			rmRemoting:    rm.GetRMRemotingInstance(),
			resourceCache: sync.Map{},
		}
	})
	return sagaResourceManagerInstance
}

func (s *SagaResourceManager) RegisterResource(resource rm.Resource) error {
	if _, ok := resource.(*SagaResource); !ok {
		return fmt.Errorf("register saga resource error, SagaResource is needed, param %v", resource)
	}
	s.resourceCache.Store(resource.GetResourceId(), resource)
	return s.rmRemoting.RegisterResource(resource)

}

func (s *SagaResourceManager) GetCachedResources() *sync.Map {
	return &s.resourceCache
}

func (s *SagaResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeSAGA
}

func (s *SagaResourceManager) BranchCommit(ctx context.Context, resource rm.BranchResource) (branch.BranchStatus, error) {
	engine := GetStateMachineEngine()
	stMaInst, err := engine.Forward(ctx, resource.Xid, nil)
	if err != nil {
		if fie, ok := exception.IsForwardInvalidException(err); ok {
			log.Printf("StateMachine forward failed, xid: %s, err: %v", resource.Xid, err)
			if isInstanceNotExists(fie.ErrCode) {
				return branch.BranchStatusPhasetwoCommitted, nil
			}
		}
		log.Printf("StateMachine forward failed, xid: %s, err: %v", resource.Xid, err)
		return branch.BranchStatusPhasetwoCommitFailedRetryable, err
	}

	status := stMaInst.Status()
	compStatus := stMaInst.CompensationStatus()

	switch {
	case status == statelang.SU && compStatus == "":
		return branch.BranchStatusPhasetwoCommitted, nil
	case compStatus == statelang.SU:
		return branch.BranchStatusPhasetwoRollbacked, nil
	case compStatus == statelang.FA || compStatus == statelang.UN:
		return branch.BranchStatusPhasetwoRollbackFailedRetryable, nil
	case status == statelang.FA && compStatus == "":
		return branch.BranchStatusPhaseoneFailed, nil
	default:
		return branch.BranchStatusPhasetwoCommitFailedRetryable, nil
	}
}

func (s *SagaResourceManager) BranchRollback(ctx context.Context, resource rm.BranchResource) (branch.BranchStatus, error) {
	engine := GetStateMachineEngine()
	stMaInst, err := engine.ReloadStateMachineInstance(ctx, resource.Xid)
	if err != nil || stMaInst == nil {
		return branch.BranchStatusPhasetwoRollbacked, nil
	}

	strategy := stMaInst.StateMachine().RecoverStrategy()
	appData := resource.ApplicationData
	isTimeoutRollback := bytes.Equal(appData, []byte{byte(message.GlobalStatusTimeoutRollbacking)}) || bytes.Equal(appData, []byte{byte(message.GlobalStatusTimeoutRollbackRetrying)})

	if strategy == statelang.Forward && isTimeoutRollback {
		log.Printf("Retry by custom recover strategy [Forward] on timeout, SAGA global[%s]", resource.Xid)
		return branch.BranchStatusPhasetwoCommitFailedRetryable, nil
	}

	stMaInst, err = engine.Compensate(ctx, resource.Xid, nil)
	if err == nil && stMaInst.CompensationStatus() == statelang.SU {
		return branch.BranchStatusPhasetwoRollbacked, nil
	}

	if fie, ok := exception.IsEngineExecutionException(err); ok {
		log.Printf("StateMachine compensate failed, xid: %s, err: %v", resource.Xid, err)
		if isInstanceNotExists(fie.ErrCode) {
			return branch.BranchStatusPhasetwoRollbacked, nil
		}
	}
	log.Printf("StateMachine compensate failed, xid: %s, err: %v", resource.Xid, err)
	return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
}

func (s *SagaResourceManager) BranchRegister(ctx context.Context, param rm.BranchRegisterParam) (int64, error) {
	return s.rmRemoting.BranchRegister(param)
}

func (s *SagaResourceManager) BranchReport(ctx context.Context, param rm.BranchReportParam) error {
	return s.rmRemoting.BranchReport(param)
}

func (s *SagaResourceManager) LockQuery(ctx context.Context, param rm.LockQueryParam) (bool, error) {
	// LockQuery is not supported for Saga resources
	return false, fmt.Errorf("LockQuery is not supported for Saga resources")
}

func (s *SagaResourceManager) UnregisterResource(resource rm.Resource) error {
	// UnregisterResource is not supported for SagaResourceManager
	return fmt.Errorf("UnregisterResource is not supported for SagaResourceManager")
}

// isInstanceNotExists checks if the error code indicates StateMachineInstanceNotExists
func isInstanceNotExists(errCode string) bool {
	return errCode == fmt.Sprintf("%v", seataErrors.StateMachineInstanceNotExists)
}
