package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/sign"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/JohnRobertFord/go-plant/internal/utils"
	"github.com/caarlos0/env/v11"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

var (
	pollInterval   = 2
	reportInterval = 10
	rateLimit      = 1
)

type (
	Metrics struct {
		Memstats    *runtime.MemStats
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

func InitConfig() (*Config, error) {
	var cfg Config
	var envCfg Config

	flag.StringVar(&cfg.URL, "a", "127.0.0.1:8080", "remote endpoint env:ADDRESS")
	flag.IntVar(&cfg.ReportInterval, "r", reportInterval, "report interval env:REPORT_INTERVAL")
	flag.IntVar(&cfg.PollInterval, "p", pollInterval, "poll interval env: POLL_INTERVAL")
	flag.IntVar(&cfg.RateLimit, "l", rateLimit, "rate limit env: RATE_LIMIT")
	flag.StringVar(&cfg.Key, "k", "", "key for creating header signature env:KEY")

	flag.Parse()

	if err := env.Parse(&envCfg); err != nil {
		return nil, fmt.Errorf("[ERR][AGENT] cant load config: %e", err)
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

	return &cfg, nil
}

func (c *Config) String() string {
	return fmt.Sprintf("[Config] URL:%s, ReportInterval:%d, PollInterval:%d, Key:%s",
		c.URL,
		c.ReportInterval,
		c.PollInterval,
		c.Key)
}

func (m *Metrics) GetMetrics() []metrics.Element {

	mtrs := make([]metrics.Element, 29)
	mtrs[0] = metrics.FormatMetric("gauge", "Alloc", m.Memstats.Alloc)
	mtrs[1] = metrics.FormatMetric("gauge", "BuckHashSys", m.Memstats.BuckHashSys)
	mtrs[2] = metrics.FormatMetric("gauge", "Frees", m.Memstats.Frees)
	mtrs[3] = metrics.FormatMetric("gauge", "GCSys", m.Memstats.GCSys)
	mtrs[4] = metrics.FormatMetric("gauge", "HeapAlloc", m.Memstats.HeapAlloc)
	mtrs[5] = metrics.FormatMetric("gauge", "HeapIdle", m.Memstats.HeapIdle)
	mtrs[6] = metrics.FormatMetric("gauge", "HeapInuse", m.Memstats.HeapInuse)
	mtrs[7] = metrics.FormatMetric("gauge", "HeapObjects", m.Memstats.HeapObjects)
	mtrs[8] = metrics.FormatMetric("gauge", "HeapReleased", m.Memstats.HeapReleased)
	mtrs[9] = metrics.FormatMetric("gauge", "HeapSys", m.Memstats.HeapSys)
	mtrs[10] = metrics.FormatMetric("gauge", "LastGC", m.Memstats.LastGC)
	mtrs[11] = metrics.FormatMetric("gauge", "Lookups", m.Memstats.Lookups)
	mtrs[12] = metrics.FormatMetric("gauge", "MCacheInuse", m.Memstats.MCacheInuse)
	mtrs[13] = metrics.FormatMetric("gauge", "MCacheSys", m.Memstats.MCacheSys)
	mtrs[14] = metrics.FormatMetric("gauge", "MSpanInuse", m.Memstats.MSpanInuse)
	mtrs[15] = metrics.FormatMetric("gauge", "MSpanSys", m.Memstats.MSpanSys)
	mtrs[16] = metrics.FormatMetric("gauge", "Mallocs", m.Memstats.Mallocs)
	mtrs[17] = metrics.FormatMetric("gauge", "NextGC", m.Memstats.NextGC)
	mtrs[18] = metrics.FormatMetric("gauge", "NumForcedGC", uint64(m.Memstats.NumForcedGC))
	mtrs[19] = metrics.FormatMetric("gauge", "NumGC", uint64(m.Memstats.NumGC))
	mtrs[20] = metrics.FormatMetric("gauge", "OtherSys", m.Memstats.OtherSys)
	mtrs[21] = metrics.FormatMetric("gauge", "PauseTotalNs", m.Memstats.PauseTotalNs)
	mtrs[22] = metrics.FormatMetric("gauge", "StackInuse", m.Memstats.StackInuse)
	mtrs[23] = metrics.FormatMetric("gauge", "StackSys", m.Memstats.StackSys)
	mtrs[24] = metrics.FormatMetric("gauge", "Sys", m.Memstats.Sys)
	mtrs[25] = metrics.FormatMetric("gauge", "TotalAlloc", m.Memstats.TotalAlloc)
	mtrs[26] = metrics.FormatFloatMetric("gauge", "GCCPUFraction", m.Memstats.GCCPUFraction)
	mtrs[27] = metrics.FormatCounter("counter", "PollCount", 1)
	mtrs[28] = metrics.FormatMetric("gauge", "RandomValue", rand.Uint64())

	return mtrs
}

func GetGopsutilMetrics() []metrics.Element {
	memory, _ := mem.VirtualMemory()
	usage, _ := cpu.Percent(1*time.Second, true)

	var out []metrics.Element
	out = append(out, metrics.FormatMetric("gauge", "TotalMemory", memory.Total))
	out = append(out, metrics.FormatMetric("gauge", "FreeMemory", memory.Free))
	for i, u := range usage {
		out = append(out, metrics.FormatFloatMetric("gauge", fmt.Sprintf("CPUutilization%d", i), u))
	}
	return out
}

func SendMetric(cfg *Config, els []metrics.Element) {

	var str string
	var err error
	ctx := context.Background()

	for _, el := range els {
		if el.MType == "gauge" {
			str = fmt.Sprintf("http://%s/update/%s/%s/%v", cfg.URL, el.MType, el.ID, *el.Value)
		} else {
			str = fmt.Sprintf("http://%s/update/%s/%s/%v", cfg.URL, el.MType, el.ID, *el.Delta)
		}
		err = utils.Retry(ctx, func() error {
			resp, err := http.Post(str, "text/plain", nil)
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
}

func SendJSONData(cfg *Config, els []metrics.Element) error {

	var err error
	var hash string
	ret, err := json.Marshal(els)
	if err != nil {
		return err
	}
	if cfg.Key != "" {
		hash, err = sign.SignString(string(ret), cfg.Key)
		if err != nil {
			return err
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
			req.Header.Add("HashSHA256", hash)
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func Worker(ctx context.Context, f func() error, delay time.Duration, count int) {
	workChannel := make(chan struct{})

	for i := 0; i < count; i++ {
		go func(c chan struct{}) {
			for {
				select {
				case <-c:
					err := f()
					if err != nil {
						log.Print(err)
					}
				case <-ctx.Done():
					return
				}
			}
		}(workChannel)
	}

	ticker := time.NewTicker(delay)
	go func() {
		for {
			select {
			case <-ticker.C:
				workChannel <- struct{}{}
			case <-ctx.Done():
				return
			}
		}
	}()

}
