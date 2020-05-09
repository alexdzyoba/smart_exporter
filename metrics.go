package main

import (
	"regexp"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// SMARTMetric wraps particular SMART metric and provides Prometheus integration
type SMARTMetric struct {
	// Regexp holds regular expression used to parse metric value from smartctl
	// output.
	Regexp *regexp.Regexp

	// Desc provides Prometheus metric descriptor.
	Desc *prometheus.Desc

	// Vals contains metric values per label (that is device). Vals are updated
	// periodiclly in a separate goroutine.
	Vals map[string]float64

	// This lock guards access to Vals map
	sync.RWMutex
}

func (m *SMARTMetric) Collect(ch chan<- prometheus.Metric) {
	// Report metric with read lock because they're updated in a separate
	// goroutine
	m.RLock()
	for label, val := range m.Vals {
		ch <- prometheus.MustNewConstMetric(
			m.Desc,
			prometheus.GaugeValue,
			val,
			label,
		)
	}
	m.RUnlock()
}

func (m *SMARTMetric) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.Desc
}
