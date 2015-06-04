# heka-promethus
[mozilla heka](https://github.com/mozilla-services/heka) output plugin which exposes an endpoint for prometheus to scrape

### Usage
Send a heka message from a decoder or filter of the following format.

```lua
{
	Logger = "prometheus_data", -- only matters for message_matcher
	Type = "prom_translated", -- only matters for message_matcher
	Payload = "", -- not examined 
	Fields = {
		type = 'Gauge', -- required
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
