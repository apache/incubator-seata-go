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

package tccbench

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"seata.apache.org/seata-go/v2/pkg/client"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/rm/tcc"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

const (
	configEnv          = "SEATA_GO_CONFIG_PATH"
	transactionTimeout = 30 * time.Second
)

var (
	initOnce sync.Once
	initErr  error
	proxy    *tcc.TCCServiceProxy

	pendingBranches sync.Map // xid -> chan error
	preparedCount   atomic.Uint64
	committedCount  atomic.Uint64
	rolledBackCount atomic.Uint64
)

type benchmarkTCCService struct{}

func (*benchmarkTCCService) Prepare(context.Context, interface{}) (bool, error) {
	preparedCount.Add(1)
	return true, nil
}

func (*benchmarkTCCService) Commit(_ context.Context, actionContext *tm.BusinessActionContext) (bool, error) {
	committedCount.Add(1)
	completeBranch(actionContext.Xid, nil)
	return true, nil
}

func (*benchmarkTCCService) Rollback(_ context.Context, actionContext *tm.BusinessActionContext) (bool, error) {
	rolledBackCount.Add(1)
	err := fmt.Errorf("unexpected rollback for xid %s", actionContext.Xid)
	completeBranch(actionContext.Xid, err)
	return true, nil
}

func (*benchmarkTCCService) GetActionName() string {
	return "SeataGoTCCBenchmark"
}

func completeBranch(xid string, err error) {
	if result, ok := pendingBranches.LoadAndDelete(xid); ok {
		result.(chan error) <- err
	}
}

func initialize() error {
	initOnce.Do(func() {
		if err := validateLoadFlags(); err != nil {
			initErr = err
			return
		}
		runtime.GOMAXPROCS(*gomaxprocsFlag)
		requestParameters.Payload = strings.Repeat("x", *payloadFlag)
		configPath := *configPathFlag
		if configPath == "" {
			initErr = fmt.Errorf("set %s to the client YAML config path", configEnv)
			return
		}

		client.InitPath(configPath)
		log.InitWithOption(*logPathFlag, log.WarnLevel)
		proxy, initErr = tcc.NewTCCServiceProxy(&benchmarkTCCService{})
	})
	return initErr
}

type transactionError struct {
	stage string
	xid   string
	cause error
}

func (e *transactionError) Error() string {
	return fmt.Sprintf("%s failed (xid=%s): %v", e.stage, e.xid, e.cause)
}

func (e *transactionError) Unwrap() error { return e.cause }

func failureStage(err error) string {
	var txErr *transactionError
	if errors.As(err, &txErr) {
		return txErr.stage
	}
	return "operation"
}

func runTransaction() error {
	var (
		txContext context.Context
		xid       string
		committed = make(chan error, 1)
		stage     = "begin"
	)

	err := tm.WithGlobalTx(context.Background(), &tm.GtxConfig{Name: "TCC benchmark", Timeout: transactionTimeout}, func(ctx context.Context) error {
		stage = "prepare"
		txContext = ctx
		xid = tm.GetXID(ctx)
		pendingBranches.Store(xid, committed)
		result, err := proxy.Prepare(ctx, requestParameters)
		if err != nil {
			return err
		}
		if accepted, ok := result.(bool); !ok || !accepted {
			return fmt.Errorf("Try did not return true: %v", result)
		}
		stage = "commit"
		return nil
	})
	if err != nil {
		if xid != "" {
			pendingBranches.Delete(xid)
		}
		return &transactionError{stage: stage, xid: xid, cause: err}
	}

	tx := tm.GetTx(txContext)
	if tx == nil || tx.TxStatus != message.GlobalStatusCommitted {
		pendingBranches.Delete(xid)
		return &transactionError{stage: "status", xid: xid, cause: fmt.Errorf("expected Committed, tx=%+v", tx)}
	}

	timer := time.NewTimer(transactionTimeout)
	defer timer.Stop()
	select {
	case err := <-committed:
		if err != nil {
			return &transactionError{stage: "callback", xid: xid, cause: err}
		}
		return nil
	case <-timer.C:
		pendingBranches.Delete(xid)
		return &transactionError{stage: "callback_timeout", xid: xid, cause: fmt.Errorf("Confirm missing after %s", transactionTimeout)}
	}
}

func TestTCCSmoke(t *testing.T) {
	if *configPathFlag == "" {
		t.Skipf("set %s to run the integration smoke test", configEnv)
	}
	if err := initialize(); err != nil {
		t.Fatal(err)
	}

	before := committedCount.Load()
	if err := runTransaction(); err != nil {
		t.Fatal(err)
	}
	if got := committedCount.Load() - before; got != 1 {
		t.Fatalf("expected one TCC commit callback, got %d", got)
	}
}

func BenchmarkTCC(b *testing.B) {
	if *configPathFlag == "" {
		b.Skipf("set %s to run the integration benchmark", configEnv)
	}
	if err := validateLoadFlags(); err != nil {
		b.Fatal(err)
	}
	if err := initialize(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	result := measure(*concurrencyFlag, 0, int64(b.N), runTransaction)
	b.StopTimer()
	b.ReportMetric(float64(result.Successes)/b.Elapsed().Seconds(), "txn/s")
	b.ReportMetric(result.SuccessLatency.P95MS, "p95-ms")
	b.ReportMetric(result.SuccessLatency.P99MS, "p99-ms")
	if result.Resources.CgroupCPUUsageCPUSeconds != nil {
		b.ReportMetric(*result.Resources.CgroupCPUUsageCPUSeconds/b.Elapsed().Seconds(), "cgroup-cpu-cores")
	}
	b.ReportMetric(float64(result.Resources.CgroupMemoryPeakBytes)/(1024*1024), "cgroup-peak-MiB")
	if err := validateSuccessfulPhase(result); err != nil {
		b.Fatal(err)
	}
}
