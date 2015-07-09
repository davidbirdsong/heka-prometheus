package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	Single    []*ConstMetric
	Summary   []*ConstSummary
	Histogram []*ConstHistogram
}

type ConstMetric struct {
	Value     float64
	ValueType string

	Name      string
	Labels    map[string]string
	Help      string
	Expires   int64
	valueType prometheus.ValueType
}

type ConstHistogram struct {
	Count    uint64
	Sum      float64
	Buckets  map[string]uint64
	_buckets map[float64]uint64

	Name    string
	Labels  map[string]string
	Help    string
	Expires int64
}

type ConstSummary struct {
	Count      uint64
	Sum        float64
	Quantiles  map[string]float64
	_quantiles map[float64]float64

	Name    string
	Labels  map[string]string
	Help    string
	Expires int64
}
