# TCC benchmark

An opt-in integration benchmark against a running Seata Server over Getty/TCP.
Each transaction has one TCC branch; Try and Confirm update counters without a
business database. It measures end-to-end TCC transactions, not network bandwidth.

## Setup

Use Go 1.20 or later and a reachable Seata Server. Run commands from the
repository root. Create a local client configuration using this example,
replacing `127.0.0.1:8091` with the Server address when needed:

```yaml
seata:
  enabled: true
  application-id: seata-go-tcc-benchmark
  tx-service-group: default_tx_group
  registry:
    type: file
  service:
    vgroup-mapping:
      default_tx_group: default
    grouplist:
      default: 127.0.0.1:8091
  transport:
    protocol: seata
    compressor: none
    enable-tm-client-batch-send-request: false
    enable-rm-client-batch-send-request: false
  tcc:
    fence:
      enable: false
```

Set the absolute path to that file:

```bash
export SEATA_GO_CONFIG_PATH=/absolute/path/to/seatago.yml
```

The client must also reach the address advertised by the Server. The smoke test
and Go benchmark skip without a configuration path; timed load requires
`-tcc.load` explicitly.

## Run

Check one transaction:

```bash
go test -run '^TestTCCSmoke$' -count=1 -v -timeout 90s ./benchmark/tcc
```

Run with four workers and GOMAXPROCS=4, warming up for 10 seconds and measuring
for 30 seconds:

```bash
mkdir -p results
go test -run '^TestTCCLoad$' -count=1 -v -timeout 5m ./benchmark/tcc -args \
  -tcc.load -tcc.concurrency=4 -tcc.gomaxprocs=4 \
  -tcc.warmup=10s -tcc.duration=30s -tcc.output=results/tcc-c4-g4-r1.json
```

The output directory must exist. Use a different output path for each run;
existing files are never overwritten. `-tcc.payload-bytes` adds an ASCII payload
(default 0), excluding protocol overhead. `-tcc.config` can override the config
path, and `-tcc.log` sets the SDK log path.

For standard Go benchmark output and allocation metrics:

```bash
go test -run '^$' -bench '^BenchmarkTCC$' -benchtime=10s -count=3 \
  -timeout 3m ./benchmark/tcc -args -tcc.concurrency=4 -tcc.gomaxprocs=4
```

## Result interpretation

- Success requires a successful Try, global status `Committed`, and matching
  Confirm. Failures, missing Confirm, or unexpected Cancel fail the run.
- Timed load separates warm-up from measurement and includes in-flight drain
  time in the TPS denominator. Worker count and GOMAXPROCS are independent.
- P95/P99 use histogram bucket upper bounds with at most 1/64 relative bucket
  width. Concurrent Go benchmark `ns/op` is elapsed time per operation, not
  individual transaction latency.
- JSON includes settings, source/runtime metadata, callback and error counts,
  latency histograms, and resource samples. Linux cgroup v2 figures cover the
  client's cgroup; process RSS covers only the client. Sampling is every 500 ms.
  Unavailable memory/RSS reads are 0; unavailable CPU usage is null.
- Keep Go, Server, configuration, and resource limits fixed when comparing runs.
  Preserve all repetitions; collect CPU profiles separately from throughput runs.
