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
	"errors"
	"flag"
	"sync"
	"time"

	"seata.apache.org/seata-go/v2/pkg/rm"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/util/fanout"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type phaseTwoContext struct {
	Xid        string
	BranchID   int64
	ResourceID string
}

type cleanupResult struct {
	cleanedByResource map[string]int
	failed            []phaseTwoContext
}

type AsyncWorkerConfig struct {
	BufferLimit            int           `yaml:"buffer_limit" json:"buffer_limit"`
	BufferCleanInterval    time.Duration `yaml:"buffer_clean_interval" json:"buffer_clean_interval"`
	ReceiveChanSize        int           `yaml:"receive_chan_size" json:"receive_chan_size"`
	CommitWorkerCount      int           `yaml:"commit_worker_count" json:"commit_worker_count"`
	CommitWorkerBufferSize int           `yaml:"commit_worker_buffer_size" json:"commit_worker_buffer_size"`
}

func (cfg *AsyncWorkerConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.BufferLimit, prefix+".buffer_size", 10000, "async worker commit buffer limit.")
	f.DurationVar(&cfg.BufferCleanInterval, prefix+".buffer.clean_interval", time.Second, "async worker commit buffer interval")
	f.IntVar(&cfg.ReceiveChanSize, prefix+".channel_size", 10000, "async worker commit channel size")
	f.IntVar(&cfg.CommitWorkerCount, prefix+".worker_count", 10, "async worker commit worker count")
	f.IntVar(&cfg.CommitWorkerBufferSize, prefix+".worker_buffer_size", 1000, "async worker commit worker buffer size")
}

// AsyncWorker executor for branch transaction commit and undo log
type AsyncWorker struct {
	conf       AsyncWorkerConfig
	batchLimit int
	jobLimit   int

	commitQueue  chan phaseTwoContext
	resourceMgr  datasource.DataSourceManager
	commitWorker *fanout.Fanout

	pendingMu         sync.Mutex
	pendingByResource map[string]int
	// A branch holds a slot from admission through cleanup, including time in commitQueue.
	// Successful cleanup or cancellation before enqueue releases the slot.
	pendingBranches  int
	pendingLimit     int
	perResourceLimit int

	branchCommitTotal          prometheus.Counter
	doBranchCommitFailureTotal prometheus.Counter
	receiveChanLength          prometheus.Gauge
	rePutBackToQueue           prometheus.Counter
}

func NewAsyncWorker(ctx context.Context, prom prometheus.Registerer, conf AsyncWorkerConfig, sourceManager datasource.DataSourceManager) *AsyncWorker {
	var asyncWorker AsyncWorker
	asyncWorker.conf = conf
	asyncWorker.batchLimit = conf.BufferLimit * 2 / 3
	if asyncWorker.batchLimit < 1 {
		asyncWorker.batchLimit = 1
	}
	asyncWorker.jobLimit = conf.CommitWorkerCount + conf.CommitWorkerBufferSize
	asyncWorker.perResourceLimit = asyncWorker.batchLimit*(asyncWorker.jobLimit+1) + conf.ReceiveChanSize
	// Reserve one batch for other resources even when one resource exhausts its capacity.
	asyncWorker.pendingLimit = asyncWorker.perResourceLimit + asyncWorker.batchLimit
	asyncWorker.pendingByResource = make(map[string]int)
	asyncWorker.commitQueue = make(chan phaseTwoContext, asyncWorker.conf.ReceiveChanSize)
	asyncWorker.resourceMgr = sourceManager
	asyncWorker.commitWorker = fanout.New("asyncWorker",
		fanout.WithWorker(asyncWorker.conf.CommitWorkerCount),
		fanout.WithBuffer(asyncWorker.conf.CommitWorkerBufferSize),
	)

	asyncWorker.branchCommitTotal = promauto.With(prom).NewCounter(prometheus.CounterOpts{
		Name: "async_worker_branch_commit_total",
		Help: "the total count of branch commit total count",
	})
	asyncWorker.doBranchCommitFailureTotal = promauto.With(prom).NewCounter(prometheus.CounterOpts{
		Name: "async_worker_branch_commit_failure_total",
		Help: "the total count of branch commit failure count",
	})
	asyncWorker.receiveChanLength = promauto.With(prom).NewGauge(prometheus.GaugeOpts{
		Name: "async_worker_receive_channel_length",
		Help: "the current length of the receive channel size",
	})
	asyncWorker.rePutBackToQueue = promauto.With(prom).NewCounter(prometheus.CounterOpts{
		Name: "async_worker_commit_failure_retry_counter",
		Help: "the counter of commit failure retry counter",
	})

	go asyncWorker.run(ctx)

	return &asyncWorker
}

// BranchCommit commit branch transaction
func (aw *AsyncWorker) BranchCommit(ctx context.Context, req rm.BranchResource) (branch.BranchStatus, error) {
	phaseCtx := phaseTwoContext{
		Xid:        req.Xid,
		BranchID:   req.BranchId,
		ResourceID: req.ResourceId,
	}

	aw.branchCommitTotal.Add(1)

	if !aw.reserveCleanupSlot(req.ResourceId) {
		aw.doBranchCommitFailureTotal.Inc()
		return branch.BranchStatusPhasetwoCommitFailedRetryable, errors.New("async undo log cleanup capacity exhausted")
	}

	select {
	case aw.commitQueue <- phaseCtx:
		if phaseCtx.ResourceID == "" {
			log.Warnf("AT undo log cleanup has no resource ID: xid=%s, branchID=%d", phaseCtx.Xid, phaseCtx.BranchID)
		}
	case <-ctx.Done():
		aw.releaseCleanupSlots(map[string]int{req.ResourceId: 1})
		return branch.BranchStatusPhasetwoCommitFailedRetryable, ctx.Err()
	}

	aw.receiveChanLength.Add(float64(len(aw.commitQueue)))

	return branch.BranchStatusPhasetwoCommitted, nil
}

func (aw *AsyncWorker) reserveCleanupSlot(resourceID string) bool {
	aw.pendingMu.Lock()
	defer aw.pendingMu.Unlock()
	if aw.pendingBranches >= aw.pendingLimit || aw.pendingByResource[resourceID] >= aw.perResourceLimit {
		return false
	}
	aw.pendingBranches++
	aw.pendingByResource[resourceID]++
	return true
}

func (aw *AsyncWorker) releaseCleanupSlots(cleanedByResource map[string]int) {
	aw.pendingMu.Lock()
	defer aw.pendingMu.Unlock()
	for resourceID, cleaned := range cleanedByResource {
		aw.pendingBranches -= cleaned
		aw.pendingByResource[resourceID] -= cleaned
		if aw.pendingByResource[resourceID] == 0 {
			delete(aw.pendingByResource, resourceID)
		}
	}
}

func (aw *AsyncWorker) run(ctx context.Context) {
	ticker := time.NewTicker(aw.conf.BufferCleanInterval)
	defer ticker.Stop()
	defer aw.commitWorker.Close()

	// Reserve a result slot per job so workers can always finish without a queue consumer.
	results := make(chan cleanupResult, aw.jobLimit)
	ready := make([]phaseTwoContext, 0, aw.batchLimit)
	var retries, dueRetries []phaseTwoContext
	// Count submitted jobs until this loop consumes their results.
	outstandingJobs := 0
	preferRetry := false
	flush := false
	for {
		newBatchReady := len(ready) > 0 && (len(ready) >= aw.batchLimit || flush)
		if outstandingJobs < aw.jobLimit && (newBatchReady || len(dueRetries) > 0) {
			var batch []phaseTwoContext
			// Alternate when both queues can run so retries and fresh work both make progress.
			if len(dueRetries) > 0 && (!newBatchReady || preferRetry) {
				batchEnd := len(dueRetries)
				if batchEnd > aw.batchLimit {
					batchEnd = aw.batchLimit
				}
				batch = append(batch, dueRetries[:batchEnd]...)
				for i := 0; i < batchEnd; i++ {
					dueRetries[i] = phaseTwoContext{}
				}
				dueRetries = dueRetries[batchEnd:]
				if len(dueRetries) == 0 {
					dueRetries = nil
				}
				preferRetry = false
			} else {
				batch = append(batch, ready...)
				ready = ready[:0]
				flush = false
				preferRetry = true
			}
			if err := aw.commitWorker.Do(ctx, func(ctx context.Context) {
				result := cleanupResult{failed: batch}
				defer func() { results <- result }()
				result = aw.cleanUndoLogs(ctx, batch)
			}); err != nil {
				aw.doBranchCommitFailureTotal.Add(1)
				log.Errorf("schedule undo log cleanup: %v", err)
				return
			}
			outstandingJobs++
			continue
		}

		intake := aw.commitQueue
		if len(ready) >= aw.batchLimit {
			intake = nil
		}
		select {
		case phaseCtx := <-intake:
			ready = append(ready, phaseCtx)
		case result := <-results:
			outstandingJobs--
			if len(result.cleanedByResource) > 0 {
				aw.releaseCleanupSlots(result.cleanedByResource)
			}
			if len(result.failed) > 0 {
				aw.rePutBackToQueue.Add(float64(len(result.failed)))
				retries = append(retries, result.failed...)
			}
		case <-ticker.C:
			if len(dueRetries) == 0 {
				// Drain one retry cohort at a time so new failures do not retain its storage.
				dueRetries, retries = retries, nil
			}
			flush = len(ready) > 0
		case <-ctx.Done():
			return
		}
	}
}

func (aw *AsyncWorker) cleanUndoLogs(ctx context.Context, phaseCtxs []phaseTwoContext) cleanupResult {
	groups := make(map[string][]phaseTwoContext)
	for _, phaseCtx := range phaseCtxs {
		groups[phaseCtx.ResourceID] = append(groups[phaseCtx.ResourceID], phaseCtx)
	}

	result := cleanupResult{cleanedByResource: make(map[string]int, len(groups))}
	for resourceID, group := range groups {
		if resourceID == "" {
			result.failed = append(result.failed, group...)
			continue
		}
		failed := aw.cleanResourceUndoLogs(ctx, resourceID, group)
		if cleaned := len(group) - len(failed); cleaned > 0 {
			result.cleanedByResource[resourceID] = cleaned
		}
		result.failed = append(result.failed, failed...)
	}
	return result
}

func (aw *AsyncWorker) cleanResourceUndoLogs(ctx context.Context, resID string, phaseCtxs []phaseTwoContext) []phaseTwoContext {
	// Bound the number of branch keys and SQL parameters in each cleanup statement.
	const undoLogDeleteBatchSize = 100

	val, ok := aw.resourceMgr.GetCachedResources().Load(resID)
	if !ok {
		return phaseCtxs
	}

	res := val.(*DBResource)
	conn, err := res.db.Conn(ctx)
	if err != nil {
		return phaseCtxs
	}
	defer conn.Close()

	undoMgr, err := undo.GetUndoLogManager(res.dbType)
	if err != nil {
		return phaseCtxs
	}

	var failed []phaseTwoContext
	for start := 0; start < len(phaseCtxs); start += undoLogDeleteBatchSize {
		end := start + undoLogDeleteBatchSize
		if end > len(phaseCtxs) {
			end = len(phaseCtxs)
		}
		batch := phaseCtxs[start:end]
		xids := make([]string, len(batch))
		branchIDs := make([]int64, len(batch))
		for i, phaseCtx := range batch {
			xids[i] = phaseCtx.Xid
			branchIDs[i] = phaseCtx.BranchID
		}
		if err := undoMgr.BatchDeleteUndoLog(xids, branchIDs, conn); err != nil {
			failed = append(failed, batch...)
		}
	}
	return failed
}
