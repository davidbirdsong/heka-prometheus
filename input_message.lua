{
	Logger = "prometheus_data", -- only matters for message_matcher
	Type = "prom_translated", -- only matters for message_matcher
	Payload = "", -- not examined 
	Fields = {
		metric_type = 'Gauge', -- required
		metric_name = 'net_counters', -- required
		metric_help = 'number of packets on network', -- required

		-- required, must be value_type=3 
		-- see https://github.com/mozilla-services/heka/blob/dev/message/message.proto#L23
		metric_value = {value_type=3, value=100}, 

		metric_expires = '70s',  -- optional, golang duration strings '100ms' etc..

		-- optional, but table lengths must match each other
		-- type is inferred by heka, but must be string
		metric_labelnames = {'host', 'service'},  
		metric_labelvalues = {'sjc1-b1-11', 'boundary'},
	}
}

