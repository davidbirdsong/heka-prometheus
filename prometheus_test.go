package prometheus

import (
	"github.com/pquerna/ffjson/ffjson"

	"testing"
	//"time"
)

func TestBasicJson(t *testing.T) {
	single := `
	{
  "single": [
    {
      "name": "counter1",
      "valuetype": "counter",
      "help": "a counter that counts stuff",
      "labels": {
        "role": "barista, shift: morning"
      }
    }
  ]
}
`

	all_in_one := `
{
  "single": [
    {
      "name": "counter1",
      "valuetype": "counter",
      "help": "a counter that counts stuff",
      "labels": {
        "role": "barista, shift: morning"
      }
    },
    {
      "name": "gauge2",
      "expires": 100,
      "valuetype": "gauge",
      "help": "the gas tank",
      "labels": {
        "car": "mine, grade: premium"
      }
    }
  ],
  "histogram": [
    {
      "name": "history1",
      "help": "history of stuff",
      "labels": {
        "period": "20th century"
      },
      "count": 1,
      "sum": 100,
      "Buckets": {
        "100": 12
      }
    }
  ],
  "summary": [
    {
      "name": "summary1",
      "help": "summary of stuff",
      "Sum": 100,
      "Count": 2,
      "Quantiles": {
        "50": 80,
        "90": 20
      }
    }
  ]
}
`
	var err error
	m := ConstMetric{}
	if err = ffjson.Unmarshal([]byte(single), &m); err != nil {
		t.Error(err)
	}

	if err = ffjson.Unmarshal([]byte(all_in_one), &m); err != nil {
		t.Error(err)
	}

}

/*
func TestBufPool(t *testing.T) {
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
*/
