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

package handler

import (
	"container/list"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"seata.apache.org/seata-go/v2/pkg/rm/tcc/fence/store/db/dao"
	"seata.apache.org/seata-go/v2/pkg/rm/tcc/fence/store/db/model"

	"seata.apache.org/seata-go/v2/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type tccFenceWrapperHandler struct {
	tccFenceDao       dao.TCCFenceStore
	logQueue          chan *model.FenceLogIdentity
	logCache          list.List
	cacheMutex        sync.Mutex
	logQueueOnce      sync.Once
	logQueueCloseOnce sync.Once
	logCacheOnce      sync.Once
	logTaskOnce       sync.Once
	db                *sql.DB
	dbMutex           sync.RWMutex
	lifecycleMutex    sync.Mutex
	stopping          bool
	destroyed         bool
	shutdownPending   int
	stopDrainCache    chan struct{}
	stopLogCleanTask  chan struct{}
	logTaskWg         sync.WaitGroup
	cleanerWg         sync.WaitGroup
	drainWg           sync.WaitGroup
}

const (
	maxQueueSize  = 500
	channelDelete = 5
	cleanExpired  = 24 * time.Hour
	// maxFenceLogCacheRetries is how many times a fence log identity may be re-queued to logCache
	// after a failed drain (Begin/delete/commit). Beyond this, entries are dropped to avoid unbounded retry.
	maxFenceLogCacheRetries = 3
)

// fenceLogCacheEntry is the value type stored in logCache for overflow and retry processing.
type fenceLogCacheEntry struct {
	identity   model.FenceLogIdentity
	retryCount int
}

var (
	fenceHandler       *tccFenceWrapperHandler
	fenceOnce          sync.Once
	cleanIntervalNanos atomic.Int64

	fenceLogCleanRetryExhaustedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tcc_fence_log_clean_retry_exhausted_total",
		Help: "Number of TCC fence log identities removed from the in-memory retry cache after cleanup retries were exhausted; database rows remain eligible for a later scan.",
	})
)

func init() {
	cleanIntervalNanos.Store(int64(5 * time.Minute))
}

func currentCleanInterval() time.Duration {
	return time.Duration(cleanIntervalNanos.Load())
}

func GetFenceHandler() *tccFenceWrapperHandler {
	if fenceHandler == nil {
		fenceOnce.Do(func() {
			fenceHandler = &tccFenceWrapperHandler{
				tccFenceDao: dao.GetTccFenceStoreDatabaseMapper(),
			}
		})
	}
	return fenceHandler
}

func (handler *tccFenceWrapperHandler) InitCleanPeriod(d time.Duration) {
	cleanIntervalNanos.Store(int64(d))
}

func (handler *tccFenceWrapperHandler) PrepareFence(ctx context.Context, tx *sql.Tx) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName

	err := handler.insertTCCFenceLog(tx, xid, branchId, actionName, enum.StatusTried)
	if err != nil {
		if mysqlError, ok := errors.Unwrap(err).(*mysql.MySQLError); ok && mysqlError.Number == 1062 {
			log.Warnf("tcc fence record already exists, idempotency rejected. xid: %s, branchId: %d", xid, branchId)
			handler.pushCleanChannel(xid, branchId)
		}
		return fmt.Errorf("insert tcc fence record errors, prepare fence failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
	}

	return nil
}

func (handler *tccFenceWrapperHandler) CommitFence(ctx context.Context, tx *sql.Tx) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId

	fenceDo, err := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	if err != nil {
		return fmt.Errorf(" commit fence method failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
	}
	if fenceDo == nil {
		return fmt.Errorf("tcc fence record not exists, commit fence method failed. xid= %s, branchId= %d", xid, branchId)
	}

	if fenceDo.Status == enum.StatusCommitted {
		log.Infof("branch transaction has already committed before. idempotency rejected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == enum.StatusRollbacked || fenceDo.Status == enum.StatusSuspended {
		// enable warn level
		log.Warnf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return fmt.Errorf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
	}

	return handler.updateFenceStatus(tx, xid, branchId, enum.StatusCommitted)
}

func (handler *tccFenceWrapperHandler) RollbackFence(ctx context.Context, tx *sql.Tx) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName

	fenceDo, err := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	if err != nil {
		return fmt.Errorf("rollback fence method failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
	}

	// record is null, mean the need suspend
	if fenceDo == nil {
		err = handler.insertTCCFenceLog(tx, xid, branchId, actionName, enum.StatusSuspended)
		if err != nil {
			return fmt.Errorf("insert tcc fence record errors, rollback fence failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
		}
		log.Infof("Insert tcc fence suspend record xid: %s, branchId: %d", xid, branchId)
		return nil
	}

	// have rollbacked or suspended
	if fenceDo.Status == enum.StatusRollbacked || fenceDo.Status == enum.StatusSuspended {
		// enable warn level
		log.Infof("Branch transaction had already rollbacked before, idempotency rejected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == enum.StatusCommitted {
		log.Warnf("Branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return fmt.Errorf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
	}

	return handler.updateFenceStatus(tx, xid, branchId, enum.StatusRollbacked)
}

func (handler *tccFenceWrapperHandler) insertTCCFenceLog(tx *sql.Tx, xid string, branchId int64, actionName string, status enum.FenceStatus) error {
	tccFenceDo := model.TCCFenceDO{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
		Status:     status,
	}
	return handler.tccFenceDao.InsertTCCFenceDO(tx, &tccFenceDo)
}

func (handler *tccFenceWrapperHandler) updateFenceStatus(tx *sql.Tx, xid string, branchId int64, status enum.FenceStatus) error {
	return handler.tccFenceDao.UpdateTCCFenceDO(tx, xid, branchId, enum.StatusTried, status)
}

func (handler *tccFenceWrapperHandler) InitLogCleanChannel(dsn string) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Warnf("failed to open database: %v", err)
		return
	}

	handler.lifecycleMutex.Lock()
	defer handler.lifecycleMutex.Unlock()
	if handler.stopping {
		_ = db.Close()
		return
	}

	handler.dbMutex.Lock()
	handler.db = db
	handler.dbMutex.Unlock()

	if handler.logQueue == nil {
		handler.logQueue = make(chan *model.FenceLogIdentity, maxQueueSize)
	}

	handler.logQueueOnce.Do(func() {
		handler.cleanerWg.Add(1)
		go func() {
			defer handler.cleanerWg.Done()
			handler.traversalCleanChannel(db)
		}()
	})

	handler.logTaskOnce.Do(func() {
		handler.stopLogCleanTask = make(chan struct{})
		handler.logTaskWg.Add(1)
		go func() {
			defer handler.logTaskWg.Done()
			handler.initLogCleanTask(db)
		}()
	})

}

func (handler *tccFenceWrapperHandler) initLogCleanTask(db *sql.DB) {

	ticker := time.NewTicker(currentCleanInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tx, err := db.Begin()
			if err != nil {
				log.Warnf("failed to begin transaction: %v", err)
				continue
			}

			expiredTime := time.Now().Add(-cleanExpired)
			identityList, err := handler.tccFenceDao.QueryTCCFenceLogIdentityByMdDate(tx, expiredTime)

			if err != nil {
				log.Warnf("failed to delete expired logs: %v", err)
				tx.Rollback()
				continue
			}

			err = tx.Commit()
			if err != nil {
				log.Errorf("failed to commit transaction: %v", err)
			}

			handler.enqueueFenceLogIdentities(identityList)
		case <-handler.stopLogCleanTask:
			return
		}
	}
}

func (handler *tccFenceWrapperHandler) enqueueFenceLogIdentities(identityList []model.FenceLogIdentity) {
	for i := range identityList {
		select {
		case handler.logQueue <- &identityList[i]:
		case <-handler.stopLogCleanTask:
			handler.reportUnprocessed(len(identityList)-i, "log_task_shutdown")
			return
		}
	}
}

func (handler *tccFenceWrapperHandler) DestroyLogCleanChannel() {
	handler.logQueueCloseOnce.Do(func() {
		handler.lifecycleMutex.Lock()
		handler.stopping = true
		handler.lifecycleMutex.Unlock()

		if handler.stopLogCleanTask != nil {
			close(handler.stopLogCleanTask)
		}
		handler.logTaskWg.Wait()

		handler.lifecycleMutex.Lock()
		if handler.logQueue != nil {
			close(handler.logQueue)
		}
		handler.lifecycleMutex.Unlock()
		handler.cleanerWg.Wait()

		handler.lifecycleMutex.Lock()
		stopDrainCache := handler.stopDrainCache
		handler.lifecycleMutex.Unlock()
		if stopDrainCache != nil {
			close(stopDrainCache)
		}
		handler.drainWg.Wait()

		handler.cacheMutex.Lock()
		unprocessed := handler.logCache.Len()
		handler.cacheMutex.Unlock()
		handler.lifecycleMutex.Lock()
		unprocessed += handler.shutdownPending
		handler.destroyed = true
		handler.lifecycleMutex.Unlock()
		if unprocessed > 0 {
			log.Warnf("event=tcc_fence_log_clean_shutdown unprocessed=%d", unprocessed)
		}

		handler.dbMutex.Lock()
		if handler.db != nil {
			_ = handler.db.Close()
			handler.db = nil
		}
		handler.dbMutex.Unlock()
	})
}

func (handler *tccFenceWrapperHandler) deleteBatchFence(tx *sql.Tx, batch []model.FenceLogIdentity) error {
	err := handler.tccFenceDao.DeleteMultipleTCCFenceLogIdentity(tx, batch)
	if err != nil {
		return fmt.Errorf("delete batch fence log failed, batch: %v: %w", batch, err)
	}
	return nil
}

func (handler *tccFenceWrapperHandler) deleteBatchInTransaction(db *sql.DB, batch []model.FenceLogIdentity) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin fence log clean transaction: %w", err)
	}

	if err = handler.deleteBatchFence(tx, batch); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("rollback fence log clean transaction: %w", rollbackErr))
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		// database/sql marks the transaction done once Commit returns. The caller
		// requeues the idempotent delete instead of attempting an ineffective rollback.
		return fmt.Errorf("commit fence log clean transaction: %w", err)
	}

	return nil
}

func (handler *tccFenceWrapperHandler) startDrainCacheTaskLocked() {
	handler.logCacheOnce.Do(func() {
		handler.stopDrainCache = make(chan struct{})
		handler.drainWg.Add(1)
		go func() {
			defer handler.drainWg.Done()
			handler.drainCacheTask()
		}()
	})
}

func (handler *tccFenceWrapperHandler) requeueBatch(batch []model.FenceLogIdentity) {
	if len(batch) == 0 {
		return
	}

	handler.lifecycleMutex.Lock()
	if handler.stopping {
		handler.lifecycleMutex.Unlock()
		handler.reportUnprocessed(len(batch), "batch_rejected_during_shutdown")
		return
	}

	handler.cacheMutex.Lock()
	for _, identity := range batch {
		handler.logCache.PushBack(&fenceLogCacheEntry{identity: identity})
	}
	handler.cacheMutex.Unlock()

	handler.startDrainCacheTaskLocked()
	handler.lifecycleMutex.Unlock()
}

func (handler *tccFenceWrapperHandler) reportUnprocessed(count int, reason string) {
	if count == 0 {
		return
	}

	handler.lifecycleMutex.Lock()
	handler.shutdownPending += count
	destroyed := handler.destroyed
	handler.lifecycleMutex.Unlock()

	if destroyed {
		log.Warnf("event=tcc_fence_log_clean_rejected unprocessed=%d reason=%s", count, reason)
	}
}

func (handler *tccFenceWrapperHandler) flushBatch(db *sql.DB, batch []model.FenceLogIdentity) {
	if len(batch) == 0 {
		return
	}

	if err := handler.deleteBatchInTransaction(db, batch); err != nil {
		log.Errorf("event=tcc_fence_log_clean_requeued batch_size=%d error=%v", len(batch), err)
		handler.requeueBatch(batch)
	}
}

func (handler *tccFenceWrapperHandler) pushCleanChannel(xid string, branchId int64) {
	fli := &model.FenceLogIdentity{
		Xid:      xid,
		BranchId: branchId,
	}
	handler.lifecycleMutex.Lock()
	if handler.stopping {
		handler.lifecycleMutex.Unlock()
		handler.reportUnprocessed(1, "push_during_shutdown")
		return
	}

	requeue := false
	select {
	case handler.logQueue <- fli:
	default:
		requeue = true
	}
	handler.lifecycleMutex.Unlock()
	if requeue {
		handler.requeueBatch([]model.FenceLogIdentity{*fli})
	}
	log.Infof("add one log to clean queue: %v ", fli)
}

func (handler *tccFenceWrapperHandler) traversalCleanChannel(db *sql.DB) {
	batch := []model.FenceLogIdentity{}

	for li := range handler.logQueue {
		batch = append(batch, *li)

		if len(batch) == channelDelete {
			handler.flushBatch(db, batch)
			batch = batch[:0]
		}
	}

	handler.flushBatch(db, batch)
}

func (handler *tccFenceWrapperHandler) drainCacheTask() {
	ticker := time.NewTicker(currentCleanInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			handler.drainCacheOnce()
		case <-handler.stopDrainCache:
			return
		}
	}

}

func (handler *tccFenceWrapperHandler) drainCacheOnce() {
	handler.cacheMutex.Lock()
	if handler.logCache.Len() == 0 {
		handler.cacheMutex.Unlock()
		return
	}

	handler.dbMutex.RLock()
	db := handler.db
	handler.dbMutex.RUnlock()
	if db == nil {
		handler.cacheMutex.Unlock()
		return
	}

	var drained []fenceLogCacheEntry
	for e := handler.logCache.Front(); e != nil; {
		next := e.Next()
		drained = append(drained, *e.Value.(*fenceLogCacheEntry))
		handler.logCache.Remove(e)
		e = next
	}
	handler.cacheMutex.Unlock()

	batch := make([]model.FenceLogIdentity, len(drained))
	for i := range drained {
		batch[i] = drained[i].identity
	}

	if err := handler.deleteBatchInTransaction(db, batch); err != nil {
		log.Errorf("event=tcc_fence_log_clean_retry_failed batch_size=%d error=%v", len(batch), err)
		handler.cacheMutex.Lock()
		for _, item := range drained {
			if item.retryCount >= maxFenceLogCacheRetries {
				fenceLogCleanRetryExhaustedTotal.Inc()
				log.Errorf("event=tcc_fence_log_clean_retry_exhausted xid=%s branch_id=%d retry_count=%d",
					item.identity.Xid, item.identity.BranchId, item.retryCount)
				continue
			}
			handler.logCache.PushBack(&fenceLogCacheEntry{
				identity:   item.identity,
				retryCount: item.retryCount + 1,
			})
		}
		handler.cacheMutex.Unlock()
	}
}
