package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

var pollInterval = 2
var reportInterval = 10

type Element struct {
	MetricType  string
	MetricName  string
	MetricValue float64
}

type Metrics struct {
	memstats    *runtime.MemStats
	PollCount   int64
	RandomValue uint64
}

func FormatMetric(type_ string, name string, value uint64) Element {
	val := float64(value)
	return Element{
		MetricType:  type_,
		MetricName:  name,
		MetricValue: val,
	}
}

func FormatFloatMetric(type_ string, name string, value float64) Element {
	return Element{
		MetricType:  type_,
		MetricName:  name,
		MetricValue: value,
	}
}

func (m Metrics) GetMetrics() []Element {
	metrics := make([]Element, 29)
	metrics[0] = FormatMetric("gauge", "Alloc", m.memstats.Alloc)
	metrics[1] = FormatMetric("gauge", "BuckHashSys", m.memstats.BuckHashSys)
	metrics[2] = FormatMetric("gauge", "Frees", m.memstats.Frees)
	metrics[3] = FormatMetric("gauge", "GCSys", m.memstats.GCSys)
	metrics[4] = FormatMetric("gauge", "HeapAlloc", m.memstats.HeapAlloc)
	metrics[5] = FormatMetric("gauge", "HeapIdle", m.memstats.HeapIdle)
	metrics[6] = FormatMetric("gauge", "HeapInuse", m.memstats.HeapInuse)
	metrics[7] = FormatMetric("gauge", "HeapObjects", m.memstats.HeapObjects)
	metrics[8] = FormatMetric("gauge", "HeapReleased", m.memstats.HeapReleased)
	metrics[9] = FormatMetric("gauge", "HeapSys", m.memstats.HeapSys)
	metrics[10] = FormatMetric("gauge", "LastGC", m.memstats.LastGC)
	metrics[11] = FormatMetric("gauge", "Lookups", m.memstats.Lookups)
	metrics[12] = FormatMetric("gauge", "MCacheInuse", m.memstats.MCacheInuse)
	metrics[13] = FormatMetric("gauge", "MCacheSys", m.memstats.MCacheSys)
	metrics[14] = FormatMetric("gauge", "MSpanInuse", m.memstats.MSpanInuse)
	metrics[15] = FormatMetric("gauge", "MSpanSys", m.memstats.MSpanSys)
	metrics[16] = FormatMetric("gauge", "Mallocs", m.memstats.Mallocs)
	metrics[17] = FormatMetric("gauge", "NextGC", m.memstats.NextGC)
	metrics[18] = FormatMetric("gauge", "NumForcedGC", uint64(m.memstats.NumForcedGC))
	metrics[19] = FormatMetric("gauge", "NumGC", uint64(m.memstats.NumGC))
	metrics[20] = FormatMetric("gauge", "OtherSys", m.memstats.OtherSys)
	metrics[21] = FormatMetric("gauge", "PauseTotalNs", m.memstats.PauseTotalNs)
	metrics[22] = FormatMetric("gauge", "StackInuse", m.memstats.StackInuse)
	metrics[23] = FormatMetric("gauge", "StackSys", m.memstats.StackSys)
	metrics[24] = FormatMetric("gauge", "Sys", m.memstats.Sys)
	metrics[25] = FormatMetric("gauge", "TotalAlloc", m.memstats.TotalAlloc)
	metrics[26] = FormatFloatMetric("gauge", "GCCPUFraction", m.memstats.GCCPUFraction)
	metrics[27] = FormatMetric("counter", "PollCount", 1)
	metrics[28] = FormatMetric("gauge", "RandomValue", rand.Uint64())

	return metrics
}

func SendMetrics(val string) {
	resp, err := http.Post("http://localhost:8080/update/"+val, "text/plain", nil)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}

func main() {

	for {
		myM := Metrics{
			memstats: &runtime.MemStats{},
		}
		runtime.ReadMemStats(myM.memstats)
		var str string
		for _, el := range myM.GetMetrics() {
			str = fmt.Sprint(el.MetricType, "/", el.MetricName, "/", el.MetricValue)
			SendMetrics(str)
		}

		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
}
