{
	Logger = "prometheus_data", -- only matters for message_matcher
	Type = "prom_translated", -- only matters for message_matcher
	Payload = "", -- no used
	Fields = {
		metric_type = 'GaugeVec', -- required
		metric_name = 'net_counters', -- required

		-- required, must be value_type=3 
		-- see https://github.com/mozilla-services/heka/blob/dev/message/message.proto#L23
		metric_value = {value_type=3, value=100}, 

		Help = "number of packets on network", -- optional
		Help = "number of packets on network", -- optional

		-- optional, but table lengths must match each other
		-- type is inferred by heka, but must be string
		metric_labelnames = {'host', 'service'},  
		metric_labelvalues = {'sjc1-b1-11', 'boundary'},
	}
}

