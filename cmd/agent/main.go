package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

var pollInterval = 2
var reportInterval = 10
var remote *string
var pollInt *int
var repInt *int

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

func FormatMetric(t string, name string, value uint64) Element {
	val := float64(value)
	return Element{
		MetricType:  t,
		MetricName:  name,
		MetricValue: val,
	}
}

func FormatFloatMetric(t string, name string, value float64) Element {
	return Element{
		MetricType:  t,
		MetricName:  name,
		MetricValue: value,
	}
}

func (m *Metrics) GetMetrics() []Element {
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

func SendMetric(val string) {
	resp, err := http.Post(val, "text/plain", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
}

func SendData(els []Element) {
	var str string
	for _, el := range els {
		str = fmt.Sprint("http://", *remote, "/update/", el.MetricType, "/", el.MetricName, "/", el.MetricValue)
		SendMetric(str)
	}
}

// func SendData(els []Element, m *sync.Mutex, t *time.Ticker) {
// 	var str string
// 	for range t.C {
// 		m.Lock()
// 		for _, el := range els {
// 			str = fmt.Sprint("http://", *remote, "/update/", el.MetricType, "/", el.MetricName, "/", el.MetricValue)
// 			SendMetric(str)
// 		}
// 		m.Unlock()
// 	}
// }

// // func UpdateMetrics(memStats runtime.MemStats, m *sync.Mutex, t *time.Ticker) {

// // 	for range t.C {
// // 		log.Println("LOL")
// // 		m.Lock()
// // 		runtime.ReadMemStats(&memStats)
// // 		m.Unlock()
// // 	}
// // }

func main() {

	remote = flag.String("a", "127.0.0.1:8080", "remote endpoint")
	repInt = flag.Int("r", reportInterval, "report interval")
	pollInt = flag.Int("p", pollInterval, "poll interval")

	ri := os.Getenv("REPORT_INTERVAL")
	pi := os.Getenv("POLL_INTERVAL")

	flag.Parse()

	if os.Getenv("ADDRESS") != "" {
		*remote = os.Getenv("ADDRESS")
	}

	var rInt int
	if ri == "" {
		rInt = *repInt
	} else {
		if r, err := strconv.Atoi(ri); err == nil {
			rInt = r
		} else {
			rInt = *repInt
		}
	}

	var pInt int
	if pi == "" {
		pInt = *pollInt
	} else {
		if p, err := strconv.Atoi(pi); err == nil {
			pInt = p
		} else {
			pInt = *pollInt
		}
	}

	myM := Metrics{
		memstats: &runtime.MemStats{},
	}

	// pollTicker := time.NewTicker(time.Duration(pInt) * time.Second)
	// reportTicker := time.NewTicker(time.Duration(rInt) * time.Second)

	// var m sync.Mutex

	// go UpdateMetrics(*myM.memstats, &m, pollTicker)
	// go SendData(myM.GetMetrics(), &m, reportTicker)

	runtime.ReadMemStats(myM.memstats)
	SendData(myM.GetMetrics())

	if pInt <= rInt {
		c := (rInt / pInt)
		delta := rInt - (c * pInt)
		for {
			for i := 0; i < c; i++ {
				time.Sleep(time.Duration(*pollInt) * time.Second)
				runtime.ReadMemStats(myM.memstats)
			}
			time.Sleep(time.Duration(delta) * time.Second)
			SendData(myM.GetMetrics())
		}
	}

}
