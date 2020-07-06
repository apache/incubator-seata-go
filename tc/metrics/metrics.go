package metrics

import (
	"sort"
)

import (
	"github.com/rcrowley/go-metrics"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/tc/event"
)

var (
	SEATA_TRANSACTION = "seata.transaction"

	NAME_KEY = "name"

	ROLE_KEY = "role"

	METER_KEY = "meter"

	STATISTIC_KEY = "statistic"

	STATUS_KEY = "status"

	ROLE_VALUE_TC = "tc"

	ROLE_VALUE_TM = "tm"

	ROLE_VALUE_RM = "rm"

	METER_VALUE_GAUGE = "gauge"

	METER_VALUE_COUNTER = "counter"

	METER_VALUE_SUMMARY = "summary"

	METER_VALUE_TIMER = "timer"

	STATISTIC_VALUE_COUNT = "count"

	STATISTIC_VALUE_TOTAL = "total"

	STATISTIC_VALUE_TPS = "tps"

	STATISTIC_VALUE_MAX = "max"

	STATISTIC_VALUE_AVERAGE = "average"

	STATUS_VALUE_ACTIVE = "active"

	STATUS_VALUE_COMMITTED = "committed"

	STATUS_VALUE_ROLLBACKED = "rollbacked"
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
	COUNTER_ACTIVE = &Counter{
		Counter: metrics.NewCounter(),
		Name:    SEATA_TRANSACTION,
		Labels: map[string]string{
			ROLE_KEY:   ROLE_VALUE_TC,
			METER_KEY:  METER_VALUE_COUNTER,
			STATUS_KEY: STATUS_VALUE_ACTIVE,
		},
	}
	COUNTER_COMMITTED = &Counter{
		Counter: metrics.NewCounter(),
		Name:    SEATA_TRANSACTION,
		Labels: map[string]string{
			ROLE_KEY:   ROLE_VALUE_TC,
			METER_KEY:  METER_VALUE_COUNTER,
			STATUS_KEY: STATUS_VALUE_COMMITTED,
		},
	}
	COUNTER_ROLLBACKED = &Counter{
		Counter: metrics.NewCounter(),
		Name:    SEATA_TRANSACTION,
		Labels: map[string]string{
			ROLE_KEY:   ROLE_VALUE_TC,
			METER_KEY:  METER_VALUE_COUNTER,
			STATUS_KEY: STATUS_VALUE_ROLLBACKED,
		},
	}
	SUMMARY_COMMITTED = &Summary{
		Meter: metrics.NewMeter(),
		Name:  SEATA_TRANSACTION,
		Labels: map[string]string{
			ROLE_KEY:   ROLE_VALUE_TC,
			METER_KEY:  METER_VALUE_SUMMARY,
			STATUS_KEY: STATUS_VALUE_COMMITTED,
		},
	}
	SUMMARY_ROLLBACKED = &Summary{
		Meter: metrics.NewMeter(),
		Name:  SEATA_TRANSACTION,
		Labels: map[string]string{
			ROLE_KEY:   ROLE_VALUE_TC,
			METER_KEY:  METER_VALUE_SUMMARY,
			STATUS_KEY: STATUS_VALUE_ROLLBACKED,
		},
	}
	TIMER_COMMITTED = &Histogram{
		Histogram: metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
		Name:      SEATA_TRANSACTION,
		Labels: map[string]string{
			ROLE_KEY:   ROLE_VALUE_TC,
			METER_KEY:  METER_VALUE_TIMER,
			STATUS_KEY: STATUS_VALUE_COMMITTED,
		},
	}
	TIMER_ROLLBACK = &Histogram{
		Histogram: metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
		Name:      SEATA_TRANSACTION,
		Labels: map[string]string{
			ROLE_KEY:   ROLE_VALUE_TC,
			METER_KEY:  METER_VALUE_TIMER,
			STATUS_KEY: STATUS_VALUE_ROLLBACKED,
		},
	}
)

type MetricsSubscriber struct {
}

func (subscriber *MetricsSubscriber) ProcessGlobalTransactionEvent() {
	for {
		gtv := <-event.EventBus.GlobalTransactionEventChannel
		switch gtv.GetStatus() {
		case meta.GlobalStatusBegin:
			COUNTER_ACTIVE.Inc(1)
			break
		case meta.GlobalStatusCommitted:
			COUNTER_ACTIVE.Dec(1)
			COUNTER_COMMITTED.Inc(1)
			SUMMARY_COMMITTED.Mark(1)
			TIMER_COMMITTED.Update(gtv.GetEndTime() - gtv.GetBeginTime())
			break
		case meta.GlobalStatusRollbacked:
			COUNTER_ACTIVE.Dec(1)
			COUNTER_ROLLBACKED.Inc(1)
			SUMMARY_ROLLBACKED.Mark(1)
			TIMER_ROLLBACK.Update(gtv.GetEndTime() - gtv.GetBeginTime())
			break
		case meta.GlobalStatusCommitFailed:
			COUNTER_ACTIVE.Dec(1)
			break
		case meta.GlobalStatusRollbackFailed:
			COUNTER_ACTIVE.Dec(1)
			break
		case meta.GlobalStatusTimeoutRollbacked:
			COUNTER_ACTIVE.Dec(1)
			break
		case meta.GlobalStatusTimeoutRollbackFailed:
			COUNTER_ACTIVE.Dec(1)
			break
		default:
			break
		}
	}
}

func init() {
	subscriber := &MetricsSubscriber{}
	go subscriber.ProcessGlobalTransactionEvent()
}
