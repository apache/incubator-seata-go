package sql

import (
	"context"
	"flag"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/util/fanout"
	"github.com/seata/seata-go/pkg/util/log"
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

func (cfg *AsyncWorkerConfig) RegisterFlags(f *flag.FlagSet) {
	f.IntVar(&cfg.BufferLimit, "async-worker.commit.buffer_size", 10000, "async worker commit buffer limit.")
	f.DurationVar(&cfg.BufferCleanInterval, "async-worker.commit.buffer.clean_interval", time.Second, "async worker commit buffer interval")
	f.IntVar(&cfg.ReceiveChanSize, "async-worker.commit.channel_size", 10000, "async worker commit channel size")
	f.IntVar(&cfg.CommitWorkerCount, "async-worker.commit.worker_count", 10, "async worker commit worker count")
	f.IntVar(&cfg.CommitWorkerBufferSize, "async-worker.commit.worker_buffer_size", 1000, "async worker commit worker buffer size")
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
func (aw *AsyncWorker) BranchCommit(ctx context.Context, req message.BranchCommitRequest) (branch.BranchStatus, error) {
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
		log.Errorf("do branch commit err:%v", err)
	}
}

func (aw *AsyncWorker) dealWithGroupedContexts(resID string, phaseCtxs []phaseTwoContext) {
	val, ok := aw.resourceMgr.GetManagedResources()[resID]
	if !ok {
		for i := range phaseCtxs {
			aw.rePutBackToQueue.Add(1)
			aw.commitQueue <- phaseCtxs[i]
		}
		return
	}

	res := val.(*DBResource)
	conn, err := res.target.Conn(context.Background())
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
