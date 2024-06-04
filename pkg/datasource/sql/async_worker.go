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
	"flag"
	"time"

	"seata.apache.org/seata-go/pkg/rm"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/util/fanout"
	"seata.apache.org/seata-go/pkg/util/log"
)

type phaseTwoContext struct {
	Xid        string
	BranchID   int64
	ResourceID string
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
	conf AsyncWorkerConfig

	commitQueue  chan phaseTwoContext
	resourceMgr  datasource.DataSourceManager
	commitWorker *fanout.Fanout

	branchCommitTotal          prometheus.Counter
	doBranchCommitFailureTotal prometheus.Counter
	receiveChanLength          prometheus.Gauge
	rePutBackToQueue           prometheus.Counter
}

func NewAsyncWorker(prom prometheus.Registerer, conf AsyncWorkerConfig, sourceManager datasource.DataSourceManager) *AsyncWorker {
	var asyncWorker AsyncWorker
	asyncWorker.conf = conf
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

	go asyncWorker.run()

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

	select {
	case aw.commitQueue <- phaseCtx:
	case <-ctx.Done():
	}

	aw.receiveChanLength.Add(float64(len(aw.commitQueue)))

	return branch.BranchStatusPhasetwoCommitted, nil
}

func (aw *AsyncWorker) run() {
	ticker := time.NewTicker(aw.conf.BufferCleanInterval)
	phaseCtxs := make([]phaseTwoContext, 0, aw.conf.BufferLimit)
	for {
		select {
		case phaseCtx := <-aw.commitQueue:
			phaseCtxs = append(phaseCtxs, phaseCtx)
			if len(phaseCtxs) >= aw.conf.BufferLimit*2/3 {
				aw.doBranchCommit(&phaseCtxs)
			}
		case <-ticker.C:
			aw.doBranchCommit(&phaseCtxs)
		}
	}
}

func (aw *AsyncWorker) doBranchCommit(phaseCtxs *[]phaseTwoContext) {
	if len(*phaseCtxs) == 0 {
		return
	}

	copyPhaseCtxs := make([]phaseTwoContext, len(*phaseCtxs))
	copy(copyPhaseCtxs, *phaseCtxs)
	*phaseCtxs = (*phaseCtxs)[:0]

	doBranchCommit := func(ctx context.Context) {
		groupCtxs := make(map[string][]phaseTwoContext, 16)
		for i := range copyPhaseCtxs {
			if copyPhaseCtxs[i].ResourceID == "" {
				continue
			}

			if _, ok := groupCtxs[copyPhaseCtxs[i].ResourceID]; !ok {
				groupCtxs[copyPhaseCtxs[i].ResourceID] = make([]phaseTwoContext, 0, 4)
			}

			ctxs := groupCtxs[copyPhaseCtxs[i].ResourceID]
			ctxs = append(ctxs, copyPhaseCtxs[i])
			groupCtxs[copyPhaseCtxs[i].ResourceID] = ctxs
		}

		for k := range groupCtxs {
			aw.dealWithGroupedContexts(k, groupCtxs[k])
		}
	}

	if err := aw.commitWorker.Do(context.Background(), doBranchCommit); err != nil {
		aw.doBranchCommitFailureTotal.Add(1)
		log.Errorf("do branch commit err:%v,phaseCtxs=%v", err, phaseCtxs)
	}
}

func (aw *AsyncWorker) dealWithGroupedContexts(resID string, phaseCtxs []phaseTwoContext) {
	val, ok := aw.resourceMgr.GetCachedResources().Load(resID)
	if !ok {
		for i := range phaseCtxs {
			aw.rePutBackToQueue.Add(1)
			aw.commitQueue <- phaseCtxs[i]
		}
		return
	}

	res := val.(*DBResource)
	conn, err := res.db.Conn(context.Background())
	if err != nil {
		for i := range phaseCtxs {
			aw.commitQueue <- phaseCtxs[i]
		}
	}

	defer conn.Close()

	undoMgr, err := undo.GetUndoLogManager(res.dbType)
	if err != nil {
		for i := range phaseCtxs {
			aw.rePutBackToQueue.Add(1)
			aw.commitQueue <- phaseCtxs[i]
		}
		return
	}

	for i := range phaseCtxs {
		phaseCtx := phaseCtxs[i]
		if err := undoMgr.BatchDeleteUndoLog([]string{phaseCtx.Xid}, []int64{phaseCtx.BranchID}, conn); err != nil {
			aw.rePutBackToQueue.Add(1)
			aw.commitQueue <- phaseCtx
		}
	}
}
