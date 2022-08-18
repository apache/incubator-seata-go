package monitor

var (

	// General Vars
	// These vars used in controllers.*
	TotalRequests = NewCounterWithLabel(
		"request_total",
		"Total number of processed requests",
		[]string{"cluster", "method", "status_code"},
	)

	ProcessingRequests = NewGaugeWithLabel(
		"request_processing",
		"Current number of processing requests",
		[]string{"cluster", "method"},
	)

	RequestLatency = NewHistogramWithLabel(
		"request_latency_ms",
		"Histogram of lantencies for requests",
		[]float64{200.0, 400.0, 600.0, 800.0, 1000.0, 1500.0, 2000.0, 2500.0, 3000.0, 5000.0, 10000.0, 20000.0, 30000.0, 45000.0, 60000.0},
		[]string{"cluster", "method"},
	)

	RealTimeRequestLatency = NewGaugeWithLabel(
		"realtime_request_latency_ms",
		"Histogram of max lantencies for requests",
		[]string{"cluster", "method"},
	)

	RealTimeRequestBodySize = NewGaugeWithLabel(
		"realtime_request_body_size",
		"Max ruquest body size of every request",
		[]string{"cluster", "method"},
	)

	//metastore
	EnvCacheLoadErrorTotal = NewCounterWithLabel(
		"metastore_env_cache_load_error_total",
		"Total number of load env data from db error.",
		[]string{"cluster"},
	)

	// Redis
	// These vars used in model.redis
	RedisTotalRequests = NewCounterWithLabel(
		"redis_request_total",
		"Total number of processed requests in redis",
		[]string{"cluster", "api_method", "redis_method"},
	)

	RedisRequestLatency = NewHistogramWithLabel(
		"redis_request_latency_ms",
		"Histogram of lantencies for requests in redis",
		[]float64{10.0, 20.0, 30.0, 40.0, 50.0, 100.0, 150.0, 200.0},
		[]string{"cluster", "api_method", "redis_method"},
	)

	// upserts
	// These vars used in model.redis
	UpsertsTotalRows = NewCounterWithLabel(
		"upserts_rows_total",
		"Total rows of written into backends",
		[]string{"cluster", "status"},
	)

	TableFromMetaStoreVersion = NewGaugeWithLabel(
		"table_sync_verison",
		"Current version of sync tables",
		[]string{"cluster", "env_db_table"},
	)

	UpsertsLantecy = NewGaugeWithLabel(
		"realtime_upserts_to_endpoint_latency_ms",
		"Histogram of max lantencies for requests",
		[]string{"cluster", "endpoint"},
	)
)
