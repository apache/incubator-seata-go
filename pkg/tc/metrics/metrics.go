package metrics

import (
	"sort"

	"github.com/rcrowley/go-metrics"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/event"
	"github.com/opentrx/seata-golang/v2/pkg/util/runtime"
)

var (
	SeataTransaction = "seata.transaction"

	NameKey = "name"

	RoleKey = "role"

	MeterKey = "meter"

	StatisticKey = "statistic"

	StatusKey = "status"

	RoleValueTc = "tc"

	RoleValueTm = "client"

	RoleValueRm = "rm"

	MeterValueGauge = "gauge"

	MeterValueCounter = "counter"

	MeterValueSummary = "summary"

	MeterValueTimer = "timer"

	StatisticValueCount = "count"

	StatisticValueTotal = "total"

	StatisticValueTps = "tps"

	StatisticValueMax = "max"

	StatisticValueAverage = "average"

	StatusValueActive = "active"

	StatusValueCommitted = "committed"

	StatusValueRollbacked = "rollbacked"
)

type Counter struct {
	metrics.Counter
	Name      string
	Labels    map[string]string
	labelKeys []string
	labelVals []string
}

func (c *Counter) SortedLabels() (keys, vals []string) {
	if c.labelKeys != nil && c.labelVals != nil {
		return c.labelKeys, c.labelVals
	}
	keys, vals = sortedLabels(c.Labels)
	c.labelKeys = keys
	c.labelVals = vals
	return
}

type Summary struct {
	metrics.Meter
	Name      string
	Labels    map[string]string
	labelKeys []string
	labelVals []string
}

func (s *Summary) SortedLabels() (keys, vals []string) {
	if s.labelKeys != nil && s.labelVals != nil {
		return s.labelKeys, s.labelVals
	}
	keys, vals = sortedLabels(s.Labels)
	s.labelKeys = keys
	s.labelVals = vals
	return
}

type Histogram struct {
	metrics.Histogram
	Name      string
	Labels    map[string]string
	labelKeys []string
	labelVals []string
}

func (h *Histogram) SortedLabels() (keys, vals []string) {
	if h.labelKeys != nil && h.labelVals != nil {
		return h.labelKeys, h.labelVals
	}
	keys, vals = sortedLabels(h.Labels)
	h.labelKeys = keys
	h.labelVals = vals
	return
}

func sortedLabels(labels map[string]string) (keys, values []string) {
	keys = make([]string, 0, len(labels))
	values = make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		values = append(values, labels[k])
	}
	return
}

var (
	CounterActive = &Counter{
		Counter: metrics.NewCounter(),
		Name:    SeataTransaction,
		Labels: map[string]string{
			RoleKey:   RoleValueTc,
			MeterKey:  MeterValueCounter,
			StatusKey: StatusValueActive,
		},
	}
	CounterCommitted = &Counter{
		Counter: metrics.NewCounter(),
		Name:    SeataTransaction,
		Labels: map[string]string{
			RoleKey:   RoleValueTc,
			MeterKey:  MeterValueCounter,
			StatusKey: StatusValueCommitted,
		},
	}
	CounterRollbacked = &Counter{
		Counter: metrics.NewCounter(),
		Name:    SeataTransaction,
		Labels: map[string]string{
			RoleKey:   RoleValueTc,
			MeterKey:  MeterValueCounter,
			StatusKey: StatusValueRollbacked,
		},
	}
	SummaryCommitted = &Summary{
		Meter: metrics.NewMeter(),
		Name:  SeataTransaction,
		Labels: map[string]string{
			RoleKey:   RoleValueTc,
			MeterKey:  MeterValueSummary,
			StatusKey: StatusValueCommitted,
		},
	}
	SummaryRollbacked = &Summary{
		Meter: metrics.NewMeter(),
		Name:  SeataTransaction,
		Labels: map[string]string{
			RoleKey:   RoleValueTc,
			MeterKey:  MeterValueSummary,
			StatusKey: StatusValueRollbacked,
		},
	}
	TimerCommitted = &Histogram{
		Histogram: metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
		Name:      SeataTransaction,
		Labels: map[string]string{
			RoleKey:   RoleValueTc,
			MeterKey:  MeterValueTimer,
			StatusKey: StatusValueCommitted,
		},
	}
	TimerRollback = &Histogram{
		Histogram: metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
		Name:      SeataTransaction,
		Labels: map[string]string{
			RoleKey:   RoleValueTc,
			MeterKey:  MeterValueTimer,
			StatusKey: StatusValueRollbacked,
		},
	}
)

type Subscriber struct {
}

func (subscriber *Subscriber) ProcessGlobalTransactionEvent() {
	for {
		gtv := <-event.EventBus.GlobalTransactionEventChannel
		switch gtv.GetStatus() {
		case apis.Begin:
			CounterActive.Inc(1)
		case apis.Committed:
			CounterActive.Dec(1)
			CounterCommitted.Inc(1)
			SummaryCommitted.Mark(1)
			TimerCommitted.Update(gtv.GetEndTime() - gtv.GetBeginTime())
		case apis.RolledBack:
			CounterActive.Dec(1)
			CounterRollbacked.Inc(1)
			SummaryRollbacked.Mark(1)
			TimerRollback.Update(gtv.GetEndTime() - gtv.GetBeginTime())
		case apis.CommitFailed:
			CounterActive.Dec(1)
		case apis.RollbackFailed:
			CounterActive.Dec(1)
		case apis.TimeoutRolledBack:
			CounterActive.Dec(1)
		case apis.TimeoutRollbackFailed:
			CounterActive.Dec(1)
		default:
			break
		}
	}
}

func init() {
	subscriber := &Subscriber{}
	runtime.GoWithRecover(func() {
		subscriber.ProcessGlobalTransactionEvent()
	}, nil)
}
