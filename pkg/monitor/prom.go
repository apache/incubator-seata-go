package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func NewCounterWithLabel(name string, help string, label []string) (m *prometheus.CounterVec) {
	return promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		}, label)
}

func NewCounter(name string, help string) (m prometheus.Counter) {
	return promauto.NewCounter(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		})
}

func NewGaugeWithLabel(name string, help string, label []string) (m *prometheus.GaugeVec) {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		}, label)
}

func NewHistogramWithLabel(name string, help string, buckets []float64, label []string) (m *prometheus.HistogramVec) {
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		}, label)
}

func NewSummaryWithLabel(name string, help string, objectives map[float64]float64, label []string) (m *prometheus.SummaryVec) {
	return promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       name,
			Help:       help,
			Objectives: objectives,
		}, label)
}
