// Package prometheus implements a heka output plugin which provides acts as a prometheus endpoint
// ready for scraping
//
// Messages that arrive via the heka router must have a carefully formatted structure, all data is conveyed in Heka Fields:  http://hekad.readthedocs.org/en/v0.9.2/message/index.html
//
// Prometheus Data types limited to: Gauge and GaugeVect
///
package prometheus

import (
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	"github.com/prometheus/client_golang/prometheus"
	//"net/http"
	//"net/url"
	"fmt"
	"net/http"
	"strings"
	"sync"
	//"time"
)

var (
	metricFieldVal  = "metric_value"
	metricFieldName = "metric_name"
	metricFieldType = "metric_type"
)
var (
	metricFieldTagK = "tags_key"
	metricFieldTagV = "tags_val"
)

type PromOutConfig struct {
	Address string
}

type PromOut struct {
	config    *PromOutConfig
	rlock     *sync.RWMutex
	gaugeVecs map[string]map[string]*prometheus.GaugeVec
	gauges    map[string]prometheus.Gauge
	inSuccess prometheus.Counter
	inFailure prometheus.Counter
	l         func(string)
}

func (p *PromOut) ConfigStruct() interface{} {
	return &PromOutConfig{
		Address: "0.0.0.0:9107",
	}
}

func (p *PromOut) Init(config interface{}) error {
	p.gaugeVecs = make(map[string]map[string]*prometheus.GaugeVec)
	p.gauges = make(map[string]prometheus.Gauge)

	p.inSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hekagateway_msg_success",
			Help: "properly formatted messages",
		},
	)

	p.inFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hekagateway_msg_failed",
			Help: "improperly formatted messages",
		},
	)

	p.config = config.(*PromOutConfig)
	p.rlock = &sync.RWMutex{}
	e := prometheus.Register(p)
	if e != nil {
		return e
	}

	http.Handle("/metrics", prometheus.Handler())
	go http.ListenAndServe(p.config.Address, nil)
	return nil

}

func (p *PromOut) Describe(ch chan<- *prometheus.Desc) {

	p.rlock.RLock()
	defer p.rlock.RUnlock()

	for _, gauge := range p.gauges {
		ch <- gauge.Desc()
	}
	for _, v := range p.gaugeVecs {
		for r, gaugeVec := range v {
			if p.l != nil {
				p.l(fmt.Sprintf("Describe: %s", r))
				p.l(r)
			}

			gaugeVec.Describe(ch)
		}
	}
	ch <- p.inSuccess.Desc()
	ch <- p.inFailure.Desc()

}
func (p *PromOut) Collect(ch chan<- prometheus.Metric) {
	p.rlock.Lock()
	defer p.rlock.Unlock()
	for _, v := range p.gaugeVecs {
		for r, gv := range v {
			if p.l != nil {
				p.l(fmt.Sprintf("Collect: %s", r))
			}

			gv.Collect(ch)
		}
	}
	for _, g := range p.gauges {
		ch <- g
	}
	ch <- p.inSuccess
	ch <- p.inFailure
}

func (p *PromOut) Run(or pipeline.OutputRunner, h pipeline.PluginHelper) (err error) {
	var (
		gv    *prometheus.GaugeVec
		g     prometheus.Gauge
		m     *pMetric
		found bool
		f     *message.Field
	)
	p.l = or.LogMessage

	for pack := range or.InChan() {

		m, err = newPMetric(pack.Message)
		if err != nil {
			or.LogError(err)
			p.inFailure.Inc()
			pack.Recycle()
			continue

		}

		switch m.mType {

		case "gaugevec":

			p.rlock.Lock()

			if f = pack.Message.FindFirstField(metricFieldTagK); f == nil {
				or.LogError(fmt.Errorf("type: %s missing mandatory field: '%s'", m.mType, metricFieldTagK))

				p.inFailure.Inc()
				pack.Recycle()
				continue
			}

			tagsKeys := f.GetValueString()
			tagsLookup := strings.Join(tagsKeys, "")

			if f = pack.Message.FindFirstField(metricFieldTagV); f == nil {
			}
			tagsVals := f.GetValueString()
			if len(tagsKeys) != len(tagsVals) {
				or.LogError(fmt.Errorf("tag fields mismatched lengths: '%s/%s'",
					strings.Join(tagsKeys, ","),
					strings.Join(tagsVals, ","),
				))

				p.inFailure.Inc()
				pack.Recycle()
				continue

			}
			gopts := m.gaugeOpts(pack.Message)

			_, found = p.gaugeVecs[m.mName]
			if !found {
				p.gaugeVecs[m.mName] = make(map[string]*prometheus.GaugeVec)
				p.gaugeVecs[m.mName][tagsLookup] = prometheus.NewGaugeVec(gopts, tagsKeys)
				gv, _ = p.gaugeVecs[m.mName][tagsLookup]

			} else {
				gv, found = p.gaugeVecs[m.mName][tagsLookup]
				if !found {
					p.gaugeVecs[m.mName][tagsLookup] = prometheus.NewGaugeVec(gopts, tagsKeys)
				}
			}

			if g, err = gv.GetMetricWithLabelValues(tagsKeys...); err != nil {
				or.LogError(err)
				p.inFailure.Inc()
				pack.Recycle()
				continue
			}
			gv.Reset()
			g.Set(m.v)
			p.inSuccess.Inc()

			p.rlock.Unlock()

		case "gauge":
			p.rlock.Lock()
			g, found = p.gauges[m.mName]
			if !found {
				p.gauges[m.mName] = prometheus.NewGauge(m.gaugeOpts(pack.Message))
				g, _ = p.gauges[m.mName]
			}
			g.Set(m.v)
			p.inSuccess.Inc()

			p.rlock.Unlock()
			//g.Set(
		default:
			or.LogError(fmt.Errorf("unsupported message Type: %s", m.mType))
			p.inFailure.Inc()
			//pack.Recycle()
			//continue

		}

		pack.Recycle()
	}
	return nil
}

func init() {
	pipeline.RegisterPlugin("PrometheusOutput", func() interface{} {
		return new(PromOut)
	})
}

func newPMetric(m *message.Message) (*pMetric, error) {

	var (
		pm = &pMetric{}
		f  *message.Field
	)
	requiredMissing := func(s string) error {
		return fmt.Errorf("missing required field: %s", s)
	}

	singleFieldWrongCount := func(s string, i int) error {
		return fmt.Errorf("required singleton field: %s, with %d vals", s, i)
	}

	if f = m.FindFirstField(metricFieldVal); f != nil {
		d := f.GetValueDouble()
		if len(d) != 1 {
			return nil, singleFieldWrongCount(metricFieldVal, len(d))
		}
		pm.v = d[0]
	} else {

		return nil, requiredMissing(metricFieldVal)

	}

	if f = m.FindFirstField(metricFieldName); f != nil {
		if s := f.GetValueString(); len(s) != 1 {
			return nil, singleFieldWrongCount(metricFieldName, len(s))

		} else {
			pm.mName = s[0]
		}

	} else {
		return nil, requiredMissing(metricFieldName)
	}

	if f = m.FindFirstField(metricFieldType); f != nil {
		if s := f.GetValueString(); len(s) != 1 {
			return nil, singleFieldWrongCount(metricFieldType, len(s))

		} else {
			pm.mType = strings.ToLower(s[0])
		}

	} else {
		return nil, requiredMissing(metricFieldType)
	}

	return pm, nil
}

type pMetric struct {
	mType, mName string
	v            float64
}

func (p *pMetric) gaugeOpts(m *message.Message) prometheus.GaugeOpts {

	var f *message.Field
	gopts := prometheus.GaugeOpts{
		Name: p.mName,
	}
	if f = m.FindFirstField("Help"); f != nil {
		gopts.Help = f.GetValueString()[0]

	}
	if f = m.FindFirstField("Namespace"); f != nil {
		gopts.Namespace = f.GetValueString()[0]

	}
	return gopts
}
