package prometheus

import (
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	"github.com/prometheus/client_golang/prometheus"
	//"net/http"
	//"net/url"
	"strings"
	"sync"
	//"time"
)

type PromOutConfig struct {
	Address     string
	MetricsPath string
}

type PromOut struct {
	config    *PromOutConfig
	rlock     *sync.RWMutex
	gaugeVecs map[string]*prometheus.GaugeVec
	gauges    map[string]prometheus.Gauge
}

func (p *PromOut) ConfigStruct() interface{} {
	return &PromOutConfig{
		Address: "0.0.0.0:9107",
	}
}

func (p *PromOut) Init(config interface{}) error {
	p.config = config.(*PromOutConfig)
	p.rlock = &sync.RWMutex{}
	return nil
}

func (p *PromOut) validMsg(m *message.Message) bool {
	/*
		switch string.ToLowerm.GetType()) {
			case
		}
	*/
	return true
}

func (p *PromOut) Describe(ch chan<- *prometheus.Desc) {

	p.rlock.RLock()
	defer p.rlock.RUnlock()

	for _, gauge := range p.gauges {
		ch <- gauge.Desc()
	}
	for _, gaugeVec := range p.gaugeVecs {
		gaugeVec.Describe(ch)
	}

}
func (p *PromOut) Collect(ch chan<- prometheus.Metric) {
	p.rlock.Lock()
	defer p.rlock.Unlock()

}

func (p *PromOut) Run(or pipeline.OutputRunner, h pipeline.PluginHelper) (err error) {
	var (
		pack         *pipeline.PipelinePack
		field        *message.Field
		seen_metrics map[string]map[string]bool
	)

	for pack = range or.InChan {
		if p.validMsg(pack.Message) {
			namespace := pack.Message
			msgType := strings.ToLower(pack.Message.GetType())
			switch strings.ToLower(pack.Message.GetType()) {
			case "gaugevecs":
			case "gauge":

			}
		}
		pack.Recycle()
	}

}
func gaugeOpts(m *message.Message) prometheus.GaugeOpts {
	return prometheus.GaugeOpts{
		Namespace: m.FindFirstField("Namespace"),
		Name:      m.FindFirstField("Name"),
		Help:      m.FindFirstField("Help"),
	}

}

func newGaugeVec(m *message.Message, opts prometheus.GaugeOpts) (*prometheus.GaugeVec, error) {

	return prometheus.NewGaugeVec(gaugeOpts(m),
		m.FindFirstField("labels"),
	)

}
func newGauge(m *message.Message) (prometheus.Gauge, error) {
}

func init() {
	pipeline.RegisterPlugin("PrometheusOutput", func() interface{} {
		return new(PromOut)
	})
}
