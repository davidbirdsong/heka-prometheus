# heka-prometheus

[mozilla heka](https://github.com/mozilla-services/heka) output plugin which exposes an endpoint for prometheus to scrape. The internals are heavily influenced by the [collectd_exporter](https://github.com/prometheus/collectd_exporter)

Metrics are created and registered with the Prometheus client using an immutable implementation of the ```Metric``` interface [NewConstMetric](http://godoc.org/github.com/prometheus/client_golang/prometheus#NewConstMetric)

### Status Expiremental
As advertised the internal message format has changed from a native Heka message to a json payload. Filters need to be able to emit 100's of metrics on ```timer_event```. The author wasn't able to grok how to encode many metric events while still providing an expressive, albeit arbitrary, tagging of each metric on a single Heka message. Sending 100's of new messages from a single filter ```timer_event``` hung the router pretty badly.

The json payload solves that and also means external stuff can be blindly forwarded too.

### Usage
Send a heka message to this output plugin w/ a json message Payload.

The top level keys are one of ```single```, ```histogram```, and ```summary``` which translate to the different Constant Metrics types in prometheus.

Lists of each are sent as a subdocument of each key.

```single``` requires the ```valuetype``` key which specifies counter or gauge.

```expires``` specifies seconds the metric should survive. Expiration is calculated by adding expires to the message timestamp (heka has timestamps.)

Metrics lacking ```expires``` inherit from the default specified in toml.

entire body example:
```json
{
    "histogram": [
        {
            "Buckets": {
                "100.1": 12
            },
            "count": 1,
            "help": "history of stuff",
            "labels": {
                "period": "20th century"
            },
            "name": "hekademo_history1",
            "sum": 100
        }
    ],
    "single": [
        {
            "help": "a counter that counts stuff",
            "labels": {
                "role": "barista", "shift": "morning"
            },
            "name": "hekademo_counter1",
            "value": 10000.123,
            "valuetype": "counter"
        },
        {
            "expires": 100,
            "help": "the gas tank",
            "value": 0.123,
            "labels": {
                "car": "mine", "grade": "premium"
            },
            "name": "hekademo_gauge2",
            "valuetype": "gauge"
        }
    ],
    "summary": [
        {
            "Count": 2,
            "Quantiles": {
                "50": 80,
                "90": 20
            },
            "Sum": 100,
            "help": "summary of stuff",
            "name": "hekademo_summary1"
        }
    ]
}
```

Add the following ```toml``` to heka:
```toml
[prometheus_out]
type = "PrometheusOutput"
message_matcher = 'Logger == "Anything"' # anything to route the message properly here
Address = "127.0.0.1:9112"
default_ttl = '15s' # applied to any metrics w/ no expires, defautls to 90s

```
curl the new prometheus in heka:
```
[david@foulplay ~]$ curl http://127.0.0.1:9112/metrics  
# [lots of prometheus boilerplate metrics suprressed]
#
#
# HELP hekagateway_msg_failed improperly formatted messages
# # TYPE hekagateway_msg_failed counter
# hekagateway_msg_failed 3
# # HELP hekagateway_msg_success properly formatted messages
# # TYPE hekagateway_msg_success counter
# hekagateway_msg_success 0
## HELP net_counters number of packets on network
## TYPE net_counters gauge
#net_counters{host="machine1",service="webapp"} 100
```
### Building
append the following to ```heka/cmake/plugin_loader.cmake```
```
hg_clone(https://bitbucket.org/ww/goautoneg default)
git_clone(https://github.com/prometheus/client_golang master)
git_clone(https://github.com/prometheus/procfs master)
git_clone(http://github.com/matttproud/golang_protobuf_extensions master)
git_clone(http://github.com/golang/protobuf master)
git_clone(https://github.com/prometheus/client_model master)
git_clone(http://github.com/beorn7/perks master)
git_clone(http://github.com/pquerna/ffjson master)

add_external_plugin(git https://github.com/davidbirdsong/heka-promethus master)
```

# Putting it All Together
a contrived example since most interesting data could come from *within* heka

### minimal toml
```toml
[HttpListenInput]
address = "0.0.0.0:8325"


[prometheus_out]
type = "PrometheusOutput"
message_matcher = 'Type == "heka.httpdata.request"'
Address = "127.0.0.1:9112"
default_ttl = '15s' # applied to any metrics w/ no expire field
```


POST data to heka's listen port
```sh
[david@foulplay ~]$  curl  -X POST  http://127.0.0.1:8325 -d '{
    "histogram": [
        {
            "Buckets": {
                "100.1": 12
            },
            "count": 1,
            "help": "history of stuff",
            "labels": {
                "period": "20th century"
            },
            "name": "hekademo_history1",
            "sum": 100
        }
    ],
    "single": [
        {
            "help": "a counter that counts stuff",
            "labels": {
                "role": "barista", "shift": "morning"
            },
            "name": "hekademo_counter1",
            "value": 10000.123,
            "valuetype": "counter"
        },
        {
            "expires": 100,
            "help": "the gas tank",
            "value": 0.123,
            "labels": {
                "car": "mine", "grade": "premium"
            },
            "name": "hekademo_gauge2",
            "valuetype": "gauge"
        }
    ],
    "summary": [
        {
            "Count": 2,
            "Quantiles": {
                "50": 80,
                "90": 20
            },
            "Sum": 100,
            "help": "summary of stuff",
            "name": "hekademo_summary1"
        }
    ]
}
'
```

scrape heka for recent data
```sh
[david@foulplay ~]$ curl  -s http://127.0.0.1:9112/metrics  | grep -B2 hekademo 
# TYPE go_goroutines gauge
go_goroutines 22
# HELP hekademo_counter1 a counter that counts stuff
# TYPE hekademo_counter1 counter
hekademo_counter1{role="barista",shift="morning"} 10000.123
# HELP hekademo_gauge2 the gas tank
# TYPE hekademo_gauge2 gauge
hekademo_gauge2{car="mine",grade="premium"} 0.123
# HELP hekademo_history1 history of stuff
# TYPE hekademo_history1 histogram
hekademo_history1_bucket{period="20th century",le="100.1"} 12
hekademo_history1_bucket{period="20th century",le="+Inf"} 1
hekademo_history1_sum{period="20th century"} 100
hekademo_history1_count{period="20th century"} 1
# HELP hekademo_summary1 summary of stuff
# TYPE hekademo_summary1 summary
hekademo_summary1{quantile="50"} 80
hekademo_summary1{quantile="90"} 20
hekademo_summary1_sum 100
hekademo_summary1_count 2
```

# TODO
- multi types ```Summary``` and ```Histogram``` might need more data validation
