# heka-promethus
[mozilla heka](https://github.com/mozilla-services/heka) output plugin which exposes an endpoint for prometheus to scrape. The internals are heavily influenced by the [collectd_exporter](https://github.com/prometheus/collectd_exporter)

Metrics are created and registered with the Prometheus client using an immutable imptementation of the ```Metric``` interface [NewConstMetric](http://godoc.org/github.com/prometheus/client_golang/prometheus#NewConstMetric)

The ```valuetype``` Heka field will serve to [type](http://godoc.org/github.com/prometheus/client_golang/prometheus#ValueType) the metric to: 
- ```GaugeValue```
- ```CounterValue```
-  ```UntypedValue```

### Usage
Send a heka message from a decoder or filter of the following format.

```lua
{
	Logger = "prometheus_data", -- only matters for message_matcher
	Type = "prom_translated", -- only matters for message_matcher
	Payload = "", -- not examined 
	Fields = {
		
		-- required, one of 'Gauge', 'Counter', or '*' 
		-- non-empty but unmatched value will convert to Prometheus's 
		valuetype = 'Gauge', 
		name = 'net_counters', -- required
		help = 'number of packets on network', -- required

		-- required, must be value_type=3 
		-- see https://github.com/mozilla-services/heka/blob/dev/message/message.proto#L23
		value = {value_type=3, value=100}, 

		expires = '70s',  -- optional, golang duration strings '100ms' etc..

		-- optional, but table lengths must match each other
		-- type is inferred by heka, but must be string
		labelnames = {'host', 'service'},  
		labelvalues = {'machine1', 'webapp'},
	}
}

```

Add the following ```toml``` to heka:
```toml
[prometheus_out]
type = "PrometheusOutput"
message_matcher = 'Logger == "generate_prom_data"'
Address = "127.0.0.1:9112"
encoder = "RstEncoder"

```
(encoder is specified for error logging only )


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

add_external_plugin(git https://github.com/davidbirdsong/heka-promethus master)
```
