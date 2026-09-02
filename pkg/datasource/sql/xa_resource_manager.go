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

package sql

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/bluele/gcache"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

var branchStatusCache gcache.Cache

type XAConnConf struct {
	XaBranchExecutionTimeout time.Duration `json:"xa_branch_execution_timeout" xml:"xa_branch_execution_timeout" koanf:"xa_branch_execution_timeout"`
}

func (cfg *XAConnConf) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.XaBranchExecutionTimeout, prefix+".xa_branch_execution_timeout", time.Minute, "Undo log table name.")
}

type XAConfig struct {
	xaConnConf       XAConnConf
	TwoPhaseHoldTime time.Duration `json:"two_phase_hold_time" yaml:"xa_two_phase_hold_time" koanf:"xa_two_phase_hold_time"`
}

func (cfg *XAConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.TwoPhaseHoldTime, prefix+".two_phase_hold_time", time.Millisecond*1000, "Undo log table name.")
	cfg.xaConnConf.RegisterFlagsWithPrefix(prefix, f)
}

func InitXA(config XAConfig) *XAResourceManager {
	xaSourceManager := &XAResourceManager{
		resourceCache: sync.Map{},
		basic:         datasource.NewBasicSourceManager(),
		rmRemoting:    rm.GetRMRemotingInstance(),
		config:        config,
	}

	xaConnTimeout = config.xaConnConf.XaBranchExecutionTimeout

	branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()

	rm.GetRmCacheInstance().RegisterResourceManager(xaSourceManager)

	go xaSourceManager.xaTwoPhaseTimeoutChecker()

	return xaSourceManager
}

type XAResourceManager struct {
	config        XAConfig
	resourceCache sync.Map
	basic         *datasource.BasicSourceManager
	rmRemoting    rm.RMRemoting
}

func (xaManager *XAResourceManager) xaTwoPhaseTimeoutChecker() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			xaManager.closeTimedOutPhaseTwoConnections(time.Now())
		}
	}
}

func (xaManager *XAResourceManager) closeTimedOutPhaseTwoConnections(now time.Time) {
	xaManager.resourceCache.Range(func(key, value any) bool {
		source, ok := value.(*DBResource)
		if !ok {
			return true
		}
		source.GetKeeper().Range(func(key, value any) bool {
			connectionXA, isConnectionXA := value.(*XAConn)
			if !isConnectionXA {
				return true
			}
			prepared, holdUntilPhaseTwo, preparedAt := connectionXA.phaseTwoTimeoutSnapshot()
			if !prepared || holdUntilPhaseTwo {
				return true
			}

			if now.Sub(preparedAt) > xaManager.config.TwoPhaseHoldTime {
				if err := connectionXA.CloseForce(); err != nil {
					log.Errorf("Force close the XA physical connection: %v", err)
				}
			}
			return true
		})
		return true
	})
}

func (xaManager *XAResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeXA
}

func (xaManager *XAResourceManager) GetCachedResources() *sync.Map {
	return &xaManager.resourceCache
}

func (xaManager *XAResourceManager) RegisterResource(res rm.Resource) error {
	xaManager.resourceCache.Store(res.GetResourceId(), res)
	return xaManager.basic.RegisterResource(res)
}

func (xaManager *XAResourceManager) UnregisterResource(resource rm.Resource) error {
	return xaManager.basic.UnregisterResource(resource)
}

func (xaManager *XAResourceManager) xaIDBuilder(xid string, branchId uint64) XAXid {
	return XaIdBuild(xid, branchId)
}

func (xaManager *XAResourceManager) finishBranch(ctx context.Context, xaID XAXid, branchResource rm.BranchResource) (*XAConn, error) {
	resource, ok := xaManager.resourceCache.Load(branchResource.ResourceId)
	if !ok {
		err := fmt.Errorf("unknown resource for XA phase two, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return nil, err
	}

	dbResource, ok := resource.(*DBResource)
	if !ok {
		err := fmt.Errorf("invalid resource for XA phase two, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return nil, err
	}

	connectionProxyXA, err := dbResource.ConnectionForXA(ctx, xaID)
	if err != nil {
		wrappedErr := fmt.Errorf(
			"get connection for XA phase two, resourceId: %s: %w",
			branchResource.ResourceId,
			err,
		)
		log.Errorf(wrappedErr.Error())
		return nil, wrappedErr
	}

	return connectionProxyXA, nil
}

func phaseTwoErrorClassifier(connection *XAConn) (xa.XAPhaseTwoErrorClassifier, bool) {
	classifier, ok := connection.xaErrorClassifier.(xa.XAPhaseTwoErrorClassifier)
	return classifier, ok
}

func (xaManager *XAResourceManager) BranchCommit(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	xaID := xaManager.xaIDBuilder(branchResource.Xid, uint64(branchResource.BranchId))
	if status, err := branchStatus(xaID.String()); err == nil {
		switch status {
		case branch.BranchStatusPhasetwoCommitted:
			return branch.BranchStatusPhasetwoCommitted, nil
		case branch.BranchStatusPhasetwoRollbacked:
			return branch.BranchStatusPhasetwoCommitFailedUnretryable, fmt.Errorf(
				"XA branch %s was already rolled back",
				xaID.String(),
			)
		}
	}

	connectionProxyXA, err := xaManager.finishBranch(ctx, xaID, branchResource)
	if err != nil {
		return branch.BranchStatusPhasetwoCommitFailedRetryable, err
	}
	wasKept := connectionProxyXA.isKept()
	if !wasKept {
		defer connectionProxyXA.Close()
	}

	if err := connectionProxyXA.xaCommit(ctx, xaID, func() {
		setBranchStatus(xaID.String(), branch.BranchStatusPhasetwoCommitted)
	}); err != nil {
		log.Errorf("commit xa, resourceId: %s, err %v", branchResource.ResourceId, err)
		if classifier, ok := phaseTwoErrorClassifier(connectionProxyXA); ok {
			switch {
			case classifier.IsAlreadyCommitted(err):
				connectionProxyXA.completePhaseTwoWith(func() {
					setBranchStatus(xaID.String(), branch.BranchStatusPhasetwoCommitted)
				})
				return branch.BranchStatusPhasetwoCommitted, nil
			case classifier.IsAlreadyRollbacked(err):
				connectionProxyXA.completePhaseTwoWith(func() {
					setBranchStatus(xaID.String(), branch.BranchStatusPhasetwoRollbacked)
				})
				return branch.BranchStatusPhasetwoCommitFailedUnretryable, err
			case classifier.IsUnretryable(err):
				if closeErr := connectionProxyXA.CloseForce(); closeErr != nil {
					log.Errorf("close XA connection after unretryable commit failure: %v", closeErr)
				}
				return branch.BranchStatusPhasetwoCommitFailedUnretryable, err
			}
		}
		return branch.BranchStatusPhasetwoCommitFailedRetryable, err
	}

	log.Infof("%s was committed", xaID.String())
	return branch.BranchStatusPhasetwoCommitted, nil
}

func (xaManager *XAResourceManager) BranchRollback(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	xaID := xaManager.xaIDBuilder(branchResource.Xid, uint64(branchResource.BranchId))
	if status, err := branchStatus(xaID.String()); err == nil {
		switch status {
		case branch.BranchStatusPhasetwoRollbacked:
			return branch.BranchStatusPhasetwoRollbacked, nil
		case branch.BranchStatusPhasetwoCommitted:
			return branch.BranchStatusPhasetwoRollbackFailedUnretryable, fmt.Errorf(
				"XA branch %s was already committed",
				xaID.String(),
			)
		}
	}

	connectionProxyXA, err := xaManager.finishBranch(ctx, xaID, branchResource)
	if err != nil {
		return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
	}
	wasKept := connectionProxyXA.isKept()
	if !wasKept {
		defer connectionProxyXA.Close()
	}

	if err = connectionProxyXA.xaRollback(ctx, xaID, func() {
		setBranchStatus(xaID.String(), branch.BranchStatusPhasetwoRollbacked)
	}); err != nil {
		log.Errorf("rollback xa, resourceId: %s, err %v", branchResource.ResourceId, err)
		if classifier, ok := phaseTwoErrorClassifier(connectionProxyXA); ok {
			switch {
			case classifier.IsAlreadyRollbacked(err):
				connectionProxyXA.completePhaseTwoWith(func() {
					setBranchStatus(xaID.String(), branch.BranchStatusPhasetwoRollbacked)
				})
				return branch.BranchStatusPhasetwoRollbacked, nil
			case classifier.IsAlreadyCommitted(err):
				connectionProxyXA.completePhaseTwoWith(func() {
					setBranchStatus(xaID.String(), branch.BranchStatusPhasetwoCommitted)
				})
				return branch.BranchStatusPhasetwoRollbackFailedUnretryable, err
			case classifier.IsUnretryable(err):
				if closeErr := connectionProxyXA.CloseForce(); closeErr != nil {
					log.Errorf("close XA connection after unretryable rollback failure: %v", closeErr)
				}
				return branch.BranchStatusPhasetwoRollbackFailedUnretryable, err
			}
		}
		return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
	}

	log.Infof("%s was rollback", xaID.String())
	return branch.BranchStatusPhasetwoRollbacked, nil
}

func (xaManager *XAResourceManager) LockQuery(ctx context.Context, param rm.LockQueryParam) (bool, error) {
	return xaManager.rmRemoting.LockQuery(param)
}

func (xaManager *XAResourceManager) BranchRegister(ctx context.Context, req rm.BranchRegisterParam) (int64, error) {
	return xaManager.rmRemoting.BranchRegister(req)
}

func (xaManager *XAResourceManager) BranchReport(ctx context.Context, param rm.BranchReportParam) error {
	return xaManager.rmRemoting.BranchReport(param)
}

func (xaManager *XAResourceManager) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType, db *sql.DB) (datasource.TableMetaCache, error) {
	return xaManager.basic.CreateTableMetaCache(ctx, resID, dbType, db)
}

func branchStatus(xaBranchXid string) (branch.BranchStatus, error) {
	tmpBranchStatus, err := branchStatusCache.GetIFPresent(xaBranchXid)
	if err != nil {
		if errors.Is(err, gcache.KeyNotFoundError) {
			return branch.BranchStatusUnknown, nil
		}
		return branch.BranchStatusUnknown, err
	}

	branchStatus, isBranchStatus := tmpBranchStatus.(branch.BranchStatus)
	if !isBranchStatus {
		return branch.BranchStatusUnknown, fmt.Errorf("branchId:%s get result isn't branch status", xaBranchXid)
	}
	return branchStatus, nil
}

func setBranchStatus(xaBranchXid string, status branch.BranchStatus) {
	branchStatusCache.Set(xaBranchXid, status)
}
