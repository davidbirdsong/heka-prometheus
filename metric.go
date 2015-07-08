package prometheus

type ConstMetrics []*ConstMetric

type ConstMetric struct {
	Expires   int64
	Value     float64
	Labels    map[string]string
	Help      string
	ValueType string
	Name      string
}
