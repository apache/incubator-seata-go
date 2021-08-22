package metrics

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/opentrx/seata-golang/v2/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type promHTTPExporter struct {
	real http.Handler
}

func (exporter *promHTTPExporter) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	exporter.real.ServeHTTP(rsp, req)

	exporter.Flush(rsp)
}

func (exporter *promHTTPExporter) Flush(writer io.Writer) {
	w := writer
	var sb strings.Builder
	tracker := make(map[string]bool)

	flushCounter(tracker, &sb, CounterActive)
	flushCounter(tracker, &sb, CounterCommitted)
	flushCounter(tracker, &sb, CounterRollbacked)

	flushHistogram(tracker, &sb, TimerCommitted)
	flushHistogram(tracker, &sb, TimerRollback)

	_, err := w.Write([]byte(sb.String()))
	if err != nil {
		log.Error(err)
	}
}

func flushHistogram(tracker map[string]bool, buf *strings.Builder, histogram *Histogram) {
	keys, vals := histogram.SortedLabels()
	labels := makeLabelStr(keys, vals)
	name := strings.ReplaceAll(histogram.Name, ".", "_")
	// min
	flushGauge(tracker, buf, name+"_min", labels, histogram.Min())
	// max
	flushGauge(tracker, buf, name+"_max", labels, histogram.Max())
}

func flushCounter(tracker map[string]bool, buf *strings.Builder, counter *Counter) {
	keys, vals := counter.SortedLabels()
	labels := makeLabelStr(keys, vals)
	name := strings.ReplaceAll(counter.Name, ".", "_")
	// type
	if !tracker[name] {
		buf.WriteString("# TYPE ")
		buf.WriteString(name)
		buf.WriteString(" counter\n")
		tracker[name] = true
	}

	// metric
	buf.WriteString(name)
	buf.WriteString("{")
	buf.WriteString(labels)
	buf.WriteString("} ")
	buf.WriteString(strconv.FormatInt(counter.Count(), 10))
	buf.WriteString("\n")
}

func flushGauge(tracker map[string]bool, buf *strings.Builder, name string, labels string, val int64) {
	// type
	if !tracker[name] {
		buf.WriteString("# TYPE ")
		buf.WriteString(name)
		buf.WriteString(" gauge\n")
		tracker[name] = true
	}

	// metric
	buf.WriteString(name)
	buf.WriteString("{")
	buf.WriteString(labels)
	buf.WriteString("} ")
	buf.WriteString(strconv.FormatInt(val, 10))
	buf.WriteString("\n")
}

// input: keys=[cluster,host] values=[app1,server2]
// output: cluster="app1",host="server"
func makeLabelStr(keys, values []string) (out string) {
	if length := len(keys); length > 0 {
		out = keys[0] + "=\"" + values[0] + "\""
		for i := 1; i < length; i++ {
			out += "," + keys[i] + "=\"" + values[i] + "\""
		}
	}
	return
}

func init() {
	promReg := prometheus.NewRegistry()
	// register process and  go metrics
	promReg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	promReg.MustRegister(prometheus.NewGoCollector())

	// export http for prometheus
	srvMux := http.NewServeMux()
	srvMux.Handle("/metrics", &promHTTPExporter{
		real: promhttp.HandlerFor(promReg, promhttp.HandlerOpts{
			DisableCompression: true,
		}),
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", 9898),
		Handler: srvMux,
	}
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Error(err)
		}
	}()
}
