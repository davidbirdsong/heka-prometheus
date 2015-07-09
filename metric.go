package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"time"
)

type Descriptor struct {
	Name    string
	Labels  map[string]string
	Help    string
	Expires int64
	expires time.Time
}
type Metrics struct {
	Single    []*ConstMetric
	Summary   []*ConstSummary
	Histogram []*ConstHistogram
}

type ConstMetric struct {
	*Descriptor
	Value     float64
	ValueType string
	valueType prometheus.ValueType
}

type ConstHistogram struct {
	*Descriptor
	Count   uint64
	Sum     float64
	Buckets map[float64]uint64
}

type ConstSummary struct {
	*Descriptor
	Count     uint64
	Sum       float64
	Quantiles map[float64]float64
}
