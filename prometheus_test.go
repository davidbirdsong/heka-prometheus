package prometheus

import (
	"testing"
	"time"
)

func TestBufPool(t *testing.T) {
	payload := `
[{
  "name": "foo_metric1",
  "help": "foo metric counts stuff",
  "value": 100,
  "valuetype": "Counter",
  "labels": {
    "service": "hooman1",
    "role": "hooman_runner1"
  }
},
{
  "name": "foo_metric2",
  "help": "foo metric counts stuff",
  "value": 200,
  "expires": 20,
  "valuetype": "Counter",
  "labels": {
    "service": "hooman-2",
    "role": "hooman_runner-2"
  }
}
]
`
	timestamp := time.Now()
	d := time.Second * 10
	hsamples, err := newHekaSampleScalar([]byte(payload), d, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	h := hsamples[0]
	if h.Expires != timestamp.Add(d) {
		t.Errorf("timestamp on first metric incorrect")
	}
	if h.metric.Name != "foo_metric1" {
		t.Errorf("metric1 name incorrect: %v\n", h.metric.Name)
	}
	h = hsamples[1]
	if h.Expires != timestamp.Add(20*time.Second) {
		t.Errorf("timestamp on second metric not picked up")
	}

	if h.metric.Name != "foo_metric2" {
		t.Errorf("metric2 name incorrect: %v\n", h.metric.Name)
	}

	//if h.desc == nil {
	t.Errorf(h.desc.String())

	//}

	payload = `
[{
  "name": "foo_metric1",
  "help": "foo metric counts stuff",
  "value": 100,
  "valuetype": "Counter",
  "labels": {
    "service": "hooman1",
    "role": "hooman_runner1"
  }
},
`
	_, err = newHekaSampleScalar([]byte(payload), d, timestamp)
	if err == nil {
		t.Errorf("invalid json should have errored")
	}

}
