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
	"sync"
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
		mapa map[string]any
		mu   sync.Mutex
	}
	Config struct {
		URL            string `env:"ADDRESS"`
		ReportInterval int    `env:"REPORT_INTERVAL"`
		PollInterval   int    `env:"POLL_INTERVAL"`
		RateLimit      int    `env:"RATE_LIMIT"`
		Key            string `env:"KEY"`
	}
	gauge   float64
	counter int64
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

	if _, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.URL = envCfg.URL
	}
	if _, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		cfg.ReportInterval = envCfg.ReportInterval
	}
	if _, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		cfg.PollInterval = envCfg.PollInterval
	}
	if _, ok := os.LookupEnv("RATE_LIMIT"); ok {
		cfg.RateLimit = envCfg.RateLimit
	}
	if _, ok := os.LookupEnv("KEY"); ok {
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
func NewStorage() *Metrics {
	var ms Metrics
	ms.mapa = make(map[string]any)
	return &ms
}
func (m *Metrics) CollectMetrics() {

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.PushMetric("Alloc", memStats.Alloc)
	m.PushMetric("BuckHashSys", memStats.BuckHashSys)
	m.PushMetric("Frees", memStats.Frees)
	m.PushMetric("GCSys", memStats.GCSys)
	m.PushMetric("HeapAlloc", memStats.HeapAlloc)
	m.PushMetric("HeapIdle", memStats.HeapIdle)
	m.PushMetric("HeapInuse", memStats.HeapInuse)
	m.PushMetric("HeapObjects", memStats.HeapObjects)
	m.PushMetric("HeapReleased", memStats.HeapReleased)
	m.PushMetric("HeapSys", memStats.HeapSys)
	m.PushMetric("LastGC", memStats.LastGC)
	m.PushMetric("Lookups", memStats.Lookups)
	m.PushMetric("MCacheInuse", memStats.MCacheInuse)
	m.PushMetric("MCacheSys", memStats.MCacheSys)
	m.PushMetric("MSpanInuse", memStats.MSpanInuse)
	m.PushMetric("MSpanSys", memStats.MSpanSys)
	m.PushMetric("Mallocs", memStats.Mallocs)
	m.PushMetric("NextGC", memStats.NextGC)
	m.PushMetric("NumForcedGC", uint64(memStats.NumForcedGC))
	m.PushMetric("NumGC", uint64(memStats.NumGC))
	m.PushMetric("OtherSys", memStats.OtherSys)
	m.PushMetric("PauseTotalNs", memStats.PauseTotalNs)
	m.PushMetric("StackInuse", memStats.StackInuse)
	m.PushMetric("StackSys", memStats.StackSys)
	m.PushMetric("Sys", memStats.Sys)
	m.PushMetric("TotalAlloc", memStats.TotalAlloc)
	m.PushFloatMetric("GCCPUFraction", memStats.GCCPUFraction)
	m.PushCounter("PollCount", 1)
	m.PushMetric("RandomValue", rand.Uint64())
}

func (m *Metrics) CollectGopsutilMetrics() {
	memory, _ := mem.VirtualMemory()
	usage, _ := cpu.Percent(1*time.Second, true)

	m.PushMetric("TotalMemory", memory.Total)
	m.PushMetric("FreeMemory", memory.Free)
	for i, u := range usage {
		m.PushFloatMetric(fmt.Sprintf("CPUutilization%d", i), u)
	}
}
func (m *Metrics) PushMetric(name string, value uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mapa[name] = gauge(value)
}
func (m *Metrics) PushFloatMetric(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mapa[name] = gauge(value)
}
func (m *Metrics) PushCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.mapa[name].(counter); ok {
		m.mapa[name] = c + counter(value)
	} else {
		m.mapa[name] = c
	}

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

func GetElements(m *Metrics) []metrics.Element {
	var out []metrics.Element
	for name, value := range m.mapa {
		switch value := value.(type) {
		case gauge:
			v := float64(value)
			out = append(out, metrics.Element{
				ID:    name,
				MType: "gauge",
				Value: &v,
			})
		case counter:
			v := int64(value)
			out = append(out, metrics.Element{
				ID:    name,
				MType: "counter",
				Delta: &v,
			})
		}
	}
	fmt.Println(out)
	return out
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

func Worker(ctx context.Context, f func() error, delay time.Duration, count int, wg *sync.WaitGroup) {
	workChannel := make(chan struct{})
	wg.Add(1)
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
				wg.Done()
				return
			}
		}
	}()

}
