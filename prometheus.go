// Package prometheus implements a heka output plugin which provides acts as a prometheus endpoint
// ready for scraping
//
// Messages that arrive via the heka router must have a carefully formatted structure, all data is conveyed in Heka Fields:  http://hekad.readthedocs.org/en/v0.9.2/message/index.html
//
// Prometheus Data types limited to: Gauge and GaugeVect
///
package prometheus

import (
	"github.com/mozilla-services/heka/pipeline"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/prometheus/client_golang/prometheus"

	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type hekaSample struct {
	desc      *prometheus.Desc
	single    *ConstMetric
	hist      *ConstHistogram
	summ      *ConstSummary
	valueType prometheus.ValueType
	expires   time.Time
}

func expires(supplied int64, defaultTTL time.Duration, timestamp time.Time) time.Time {
	if supplied != 0 {

		return time.Unix(
			0, timestamp.UnixNano(),
		).Add(time.Duration(
			supplied * 1e9,
		))

	} else {
		return time.Unix(
			0, timestamp.UnixNano(),
		).Add(defaultTTL)
	}

}

func newHekaSampleScalar(payload []byte, defaultTTL time.Duration, timestamp time.Time) ([]*hekaSample, error) {
	var (
		cmetrics Metrics
		err      error
	)
	hsamples := make([]*hekaSample, 0)

	if err = ffjson.Unmarshal(payload, &cmetrics); err != nil {
		return hsamples, err
	}
	for _, c := range cmetrics.Single {
		h := &hekaSample{
			single: c,
			desc: prometheus.NewDesc(
				c.Name, c.Help, []string{},
				c.Labels,
			),
			expires: expires(c.Expires, defaultTTL, timestamp),
		}

		switch strings.ToLower(c.ValueType) {

		case "gauge":
			c.valueType = prometheus.GaugeValue
		case "counter":
			c.valueType = prometheus.CounterValue
		default:
			c.valueType = prometheus.UntypedValue
		}
		hsamples = append(hsamples, h)
	}
	var f float64
	for _, c := range cmetrics.Summary {
		c._quantiles = make(map[float64]float64)
		h := &hekaSample{
			summ: c,
			desc: prometheus.NewDesc(
				c.Name, c.Help, []string{},
				c.Labels,
			),

			expires: expires(c.Expires, defaultTTL, timestamp),
		}

		for k, v := range c.Quantiles {

			f, err = strconv.ParseFloat(k, 64)
			if err != nil {
				continue
			}
			c._quantiles[f] = v

		}
		hsamples = append(hsamples, h)
	}

	for _, c := range cmetrics.Histogram {
		c._buckets = make(map[float64]uint64)

		for k, v := range c.Buckets {
			f, err = strconv.ParseFloat(k, 64)
			if err != nil {
				continue
			}
			c._buckets[f] = v
		}

		h := &hekaSample{
			hist: c,
			desc: prometheus.NewDesc(
				c.Name, c.Help, []string{},
				c.Labels,
			),
			expires: expires(c.Expires, defaultTTL, timestamp),
		}
		hsamples = append(hsamples, h)

	}

	return hsamples, nil
}

type PromOutConfig struct {
	Address    string
	DefaultTTL string `toml:"default_ttl"`
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
	var (
		m   prometheus.Metric
		err error
	)
	for _, s := range samples {
		if now.After(s.expires) {
			continue
		}

		if s.single != nil {

			m, err = prometheus.NewConstMetric(
				s.desc, s.single.valueType, s.single.Value,
			)
			if err != nil {

				if p.errLogger != nil {
					p.errLogger(err)
				}
				continue
			}
		} else if s.hist != nil {
			m, err = prometheus.NewConstHistogram(
				s.desc, s.hist.Count,
				s.hist.Sum,
				s.hist._buckets,
			)
			if err != nil {

				if p.errLogger != nil {
					p.errLogger(err)
				}
				continue
			}

		} else if s.summ != nil {
			m, err = prometheus.NewConstSummary(
				s.desc, s.summ.Count,
				s.summ.Sum,
				s.summ._quantiles,
			)

		}

		ch <- m
	}
}

func (p *PromOut) Run(or pipeline.OutputRunner, ph pipeline.PluginHelper) (err error) {
	var (
		running  bool = true
		pack     *pipeline.PipelinePack
		hsamples []*hekaSample
	)

	ticker := time.NewTicker(time.Minute).C
	for running {
		select {
		case pack, running = <-or.InChan():
			if !running {
				continue
			}

			payload := []byte(pack.Message.GetPayload())
			msgTime := time.Unix(0, pack.Message.GetTimestamp())
			hsamples, err = newHekaSampleScalar(
				payload, p.defaultDuration, msgTime,
			)
			if err == nil {
				p.rlock.Lock()
				for _, h := range hsamples {
					p.samples[h.desc.String()] = h
					p.inSuccess.Inc()
				}
				p.rlock.Unlock()

			} else {
				or.LogError(fmt.Errorf("%v message\n<msg>\n%s\n</msg>", err, payload))

				p.inFailure.Inc()
			}

			pack.Recycle()

		case <-ticker:
			// clearn up expired samples
		}

		now := time.Now()
		p.rlock.Lock()
		for k, s := range p.samples {
			if now.After(s.expires) {
				delete(p.samples, k)
			}
		}
		p.rlock.Unlock()

	}
	return nil
}

func init() {
	pipeline.RegisterPlugin("PrometheusOutput", func() interface{} {
		return new(PromOut)
	})
}
