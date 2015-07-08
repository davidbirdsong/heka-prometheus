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
	"github.com/pquerna/ffjson/ffjson"
	"github.com/prometheus/client_golang/prometheus"

	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type hekaSample struct {
	metric  *ConstMetric
	Expires time.Time
	desc    *prometheus.Desc
	Type    prometheus.ValueType
}

func newHekaSamplesScalar(m *message.Message, defaultTTL time.Duration) ([]*hekaSample, error) {
	var (
		cmetrics ConstMetrics
		err      error
	)
	hsamples := make([]*hekaSample)

	if err = ffjson.Unmarshal([]byte(m.GetPayload()), &cmetrics); err != nil {
		return hsamples, err
	}
	for _, c := range cmetrics {
		h := new(hekaSample)
		h.metric = c

		if c.Expires != 0 {
			h.Expires = time.Unix(0,
				m.GetTimestamp()).Add(time.Duration(c.Expires * 1e9))

		} else {
			h.Expires = time.Unix(0, m.GetTimestamp()).Add(defaultTTL)

		}

		h.Expires = time.Unix(0, m.GetTimestamp()).Add(c.Expires)
		h.desc = prometheus.NewDesc(c.Name, c.Help, []string{}, c.Labels)

		switch strings.ToLower(c.ValueType) {

		case "gauge":
			h.Type = prometheus.GaugeValue
		case "counter":
			h.Type = prometheus.CounterValue
		default:
			h.Type = prometheus.UntypedValue
		}
		append(hsamples, h)

	}
	return hsamples, nil
}

/*
func newHekaSample(m *message.Message, defaultTTL time.Duration) (*hekaSample, error) {
	h := new(hekaSample)
	returnEnforceSingleString := func(n string, required bool) (string, error) {

		if f := m.FindFirstField(n); f != nil {
			if s := f.GetValueString(); len(s) != 1 {
				return "", fmt.Errorf("required singleton field: %s, with %d vals", s, len(s))

			} else {
				return s[0], nil
			}

		} else if required {
			return "", fmt.Errorf("missing required field: %s", n)
		} else {
			return "", nil
		}

	}
	var (
		mtype string
		err   error
	)
	mtype, err = returnEnforceSingleString(fieldType, true)
	if err != nil {
		return nil, err
	}

	h.Name, err = returnEnforceSingleString(fieldName, true)
	if err != nil {
		return nil, err
	}
	h.Help, err = returnEnforceSingleString(fieldHelp, true)
	if err != nil {
		return nil, err
	}

	if f := m.FindFirstField(fieldExpires); f != nil && len(f.GetValueInteger()) > 0 {
		h.Expires = time.Unix(0, f.GetValueInteger()[0])

	} else {

	}

	if f := m.FindFirstField(fieldVal); f != nil && len(f.GetValueDouble()) == 1 {
		h.Value = f.GetValueDouble()[0]

	} else {
		return nil, fmt.Errorf("single, required field invalid: %s", fieldVal)

	}

	lNameField := m.FindFirstField(fieldLabelnames)
	lValField := m.FindFirstField(fieldLabelvalues)
	if lNameField != nil {
		if lValField == nil {
			return nil, fmt.Errorf("missing label values for label names field")

		}
		if lNameField.GetValueType() != message.Field_STRING ||
			lValField.GetValueType() != message.Field_STRING {
			return nil, fmt.Errorf("incorrect data type in label fields")
		}
		names := lNameField.GetValueString()
		vals := lValField.GetValueString()
		if len(names) != len(vals) {
			return nil, fmt.Errorf("mismatched name value label field")
		}
		h.Labels = make(map[string]string)
		for i, n := range names {
			h.Labels[n] = vals[i]
		}
	}

	h.desc = prometheus.NewDesc(h.Name, h.Help, []string{}, h.Labels)

	return h, nil

}
*/

type PromOutConfig struct {
	Address    string
	DefaultTTL string
}

type PromOut struct {
	config  *PromOutConfig
	ch      chan *hekaSample
	rlock   *sync.RWMutex
	samples map[string]*hekaSample

	inSuccess       prometheus.Counter
	inFailure       prometheus.Counter
	errLogger       func(error)
	defaultDuration time.Duration
}

func (p *PromOut) ConfigStruct() interface{} {
	return &PromOutConfig{
		Address:    "0.0.0.0:9107",
		DefaultTTL: "90s",
	}
}

func (p *PromOut) Init(config interface{}) error {
	p.inSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hekagateway_msg_success",
			Help: "properly formatted messages",
		},
	)
	p.samples = make(map[string]*hekaSample)

	p.inFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hekagateway_msg_failed",
			Help: "improperly formatted messages",
		},
	)

	p.config = config.(*PromOutConfig)

	var err error
	p.defaultDuration, err = time.ParseDuration(p.config.DefaultTTL)
	if err != nil {
		return err
	}
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
	ch <- p.inSuccess.Desc()
	ch <- p.inFailure.Desc()
	defer p.rlock.RUnlock()

}
func (p *PromOut) Collect(ch chan<- prometheus.Metric) {
	ch <- p.inSuccess
	ch <- p.inFailure

	samples := make([]*hekaSample, 0, len(p.samples))
	p.rlock.RLock()
	for _, s := range p.samples {
		samples = append(samples, s)
	}
	p.rlock.RUnlock()

	now := time.Now()
	for _, s := range samples {
		if now.After(s.Expires) {
			continue
		}
		m, err := prometheus.NewConstMetric(s.desc, s.Type, s.metric.Value)
		if err != nil {

			if p.errLogger != nil {
				p.errLogger(err)
			}
			continue
		}
		ch <- m
	}
}

func (p *PromOut) Run(or pipeline.OutputRunner, ph pipeline.PluginHelper) (err error) {
	var (
		running    bool = true
		pack       *pipeline.PipelinePack
		hsamples   []*hekaSample
		metricType string
	)

	ticker := time.NewTicker(time.Minute).C
	for running {
		select {
		case pack, running = <-or.InChan():
			if !running {
				continue
			}

			metricType = "scalar"

			if f := pack.Message.FindFirstField("metricType"); f != nil {
				if s := f.GetValueString(); len(s) != 1 {
					metricType == s
				}
			}
			switch string.ToLower(metricType) {
			case "scalar", "":
				h, err = newHekaSampleScalar(pack.Message, p.defaultDuration)
			default:
				or.LogError("unsupported metricType: %s", metricType)
				continue

				p.inFailure.Inc()
				pack.Recycle()

			}

			hsamples, err = newHekaSampleScalar(pack.Message, p.defaultDuration)
			if err == nil {
				p.rlock.Lock()
				for _, h := range hsamples {
					or.LogMessage(h.desc.String())
					p.samples[h.desc.String()] = h
					p.inSuccess.Inc()
				}
				p.rlock.Unlock()

			} else {
				b, _ := or.Encode(pack)
				or.LogError(fmt.Errorf("%v message\n<msg>\n%s\n</msg>", err, b))
				p.inFailure.Inc()
			}

			pack.Recycle()

		case <-ticker:
			// clearn up expired samples
			now := time.Now()
			p.rlock.Lock()
			for k, sample := range p.samples {
				if now.After(sample.Expires) {
					delete(p.samples, k)
				}
			}
			p.rlock.Unlock()

		}

	}
	return nil
}

func init() {
	pipeline.RegisterPlugin("PrometheusOutput", func() interface{} {
		return new(PromOut)
	})
}
