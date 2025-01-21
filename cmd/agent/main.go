package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/sign"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/JohnRobertFord/go-plant/internal/utils"
	"github.com/caarlos0/env/v11"
)

var (
	pollInterval   = 2
	reportInterval = 10
	rateLimit      = 1
)

type (
	Metrics struct {
		memstats    *runtime.MemStats
		PollCount   int64
		RandomValue uint64
	}
	Config struct {
		URL            string `env:"ADDRESS"`
		ReportInterval int    `env:"REPORT_INTERVAL"`
		PollInterval   int    `env:"POLL_INTERVAL"`
		RateLimit      int    `env:"RATE_LIMIT"`
		Key            string `env:"KEY"`
	}
)

func (c *Config) String() string {
	return fmt.Sprintf("[Config] URL:%s, ReportInterval:%d, PollInterval:%d, Key:%s",
		c.URL,
		c.ReportInterval,
		c.PollInterval,
		c.Key)
}

func (m *Metrics) GetMetrics() []metrics.Element {
	mtrs := make([]metrics.Element, 29)
	mtrs[0] = metrics.FormatMetric("gauge", "Alloc", m.memstats.Alloc)
	mtrs[1] = metrics.FormatMetric("gauge", "BuckHashSys", m.memstats.BuckHashSys)
	mtrs[2] = metrics.FormatMetric("gauge", "Frees", m.memstats.Frees)
	mtrs[3] = metrics.FormatMetric("gauge", "GCSys", m.memstats.GCSys)
	mtrs[4] = metrics.FormatMetric("gauge", "HeapAlloc", m.memstats.HeapAlloc)
	mtrs[5] = metrics.FormatMetric("gauge", "HeapIdle", m.memstats.HeapIdle)
	mtrs[6] = metrics.FormatMetric("gauge", "HeapInuse", m.memstats.HeapInuse)
	mtrs[7] = metrics.FormatMetric("gauge", "HeapObjects", m.memstats.HeapObjects)
	mtrs[8] = metrics.FormatMetric("gauge", "HeapReleased", m.memstats.HeapReleased)
	mtrs[9] = metrics.FormatMetric("gauge", "HeapSys", m.memstats.HeapSys)
	mtrs[10] = metrics.FormatMetric("gauge", "LastGC", m.memstats.LastGC)
	mtrs[11] = metrics.FormatMetric("gauge", "Lookups", m.memstats.Lookups)
	mtrs[12] = metrics.FormatMetric("gauge", "MCacheInuse", m.memstats.MCacheInuse)
	mtrs[13] = metrics.FormatMetric("gauge", "MCacheSys", m.memstats.MCacheSys)
	mtrs[14] = metrics.FormatMetric("gauge", "MSpanInuse", m.memstats.MSpanInuse)
	mtrs[15] = metrics.FormatMetric("gauge", "MSpanSys", m.memstats.MSpanSys)
	mtrs[16] = metrics.FormatMetric("gauge", "Mallocs", m.memstats.Mallocs)
	mtrs[17] = metrics.FormatMetric("gauge", "NextGC", m.memstats.NextGC)
	mtrs[18] = metrics.FormatMetric("gauge", "NumForcedGC", uint64(m.memstats.NumForcedGC))
	mtrs[19] = metrics.FormatMetric("gauge", "NumGC", uint64(m.memstats.NumGC))
	mtrs[20] = metrics.FormatMetric("gauge", "OtherSys", m.memstats.OtherSys)
	mtrs[21] = metrics.FormatMetric("gauge", "PauseTotalNs", m.memstats.PauseTotalNs)
	mtrs[22] = metrics.FormatMetric("gauge", "StackInuse", m.memstats.StackInuse)
	mtrs[23] = metrics.FormatMetric("gauge", "StackSys", m.memstats.StackSys)
	mtrs[24] = metrics.FormatMetric("gauge", "Sys", m.memstats.Sys)
	mtrs[25] = metrics.FormatMetric("gauge", "TotalAlloc", m.memstats.TotalAlloc)
	mtrs[26] = metrics.FormatFloatMetric("gauge", "GCCPUFraction", m.memstats.GCCPUFraction)
	mtrs[27] = metrics.FormatCounter("counter", "PollCount", 1)
	mtrs[28] = metrics.FormatMetric("gauge", "RandomValue", rand.Uint64())

	return mtrs
}

func SendMetric(val string) {

	var err error
	ctx := context.Background()
	err = utils.Retry(ctx, func() error {
		resp, err := http.Post(val, "text/plain", nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func PrepareData(cfg *Config, els []metrics.Element) {
	var str string
	for _, el := range els {
		if el.MType == "gauge" {
			str = fmt.Sprintf("http://%s/update/%s/%s/%v", cfg.URL, el.MType, el.ID, *(el.Value))
		} else {
			str = fmt.Sprintf("http://%s/update/%s/%s/%v", cfg.URL, el.MType, el.ID, *(el.Delta))
		}
		SendMetric(str)
	}
}

func SendJSONData(cfg *Config, els []metrics.Element) {

	var err error
	var hash string
	ret, err := json.Marshal(els)
	if err != nil {
		log.Println(err)
		return
	}
	if cfg.Key != "" {
		hash, err = sign.SignString(string(ret), cfg.Key)
		if err != nil {
			log.Println(err)
		}
	}
	r := bytes.NewReader(ret)
	ctx := context.Context(context.Background())
	err = utils.Retry(ctx, func() error {
		client := &http.Client{}
		req, err := http.NewRequest("POST", "http://"+cfg.URL+"/update/", r)
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/json")
		if hash != "" {
			req.Header.Add("Hash", hash)
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func main() {
	var cfg Config
	var envCfg Config

	flag.StringVar(&cfg.URL, "a", "127.0.0.1:8080", "remote endpoint env:ADDRESS")
	flag.IntVar(&cfg.ReportInterval, "r", reportInterval, "report interval env:REPORT_INTERVAL")
	flag.IntVar(&cfg.PollInterval, "p", pollInterval, "poll interval env: POLL_INTERVAL")
	flag.IntVar(&cfg.RateLimit, "l", rateLimit, "rate limit env: RATE_LIMIT")
	flag.StringVar(&cfg.Key, "k", "", "key for creating header signature env:KEY")

	flag.Parse()

	if err := env.Parse(&envCfg); err != nil {
		log.Printf("[ERR][AGENT] cant load config: %e", err)
	}

	if os.Getenv("ADDRESS") != "" {
		cfg.URL = envCfg.URL
	}
	if os.Getenv("REPORT_INTERVAL") != "" {
		cfg.ReportInterval = envCfg.ReportInterval
	}
	if os.Getenv("POLL_INTERVAL") != "" {
		cfg.PollInterval = envCfg.PollInterval
	}
	if os.Getenv("RATE_LIMIT") != "" {
		cfg.RateLimit = envCfg.RateLimit
	}
	if os.Getenv("KEY") != "" {
		cfg.Key = envCfg.Key
	}

	myM := Metrics{
		memstats: &runtime.MemStats{},
	}

	runtime.ReadMemStats(myM.memstats)
	PrepareData(&cfg, myM.GetMetrics())
	SendJSONData(&cfg, myM.GetMetrics())

	if cfg.PollInterval <= cfg.ReportInterval {
		c := (cfg.ReportInterval / cfg.PollInterval)
		delta := cfg.ReportInterval - (c * cfg.PollInterval)
		for {
			for i := 0; i < c; i++ {
				time.Sleep(time.Duration(cfg.PollInterval) * time.Second)
				runtime.ReadMemStats(myM.memstats)
			}
			time.Sleep(time.Duration(delta) * time.Second)
			PrepareData(&cfg, myM.GetMetrics())
			SendJSONData(&cfg, myM.GetMetrics())
		}
	}
}
