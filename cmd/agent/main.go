package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/agent"
)

func main() {

	cfg, err := agent.InitConfig()
	if err != nil {
		log.Fatalf("can't init config: %e", err)
	}
	fmt.Println(cfg.String())

	myM := agent.Metrics{
		Memstats: &runtime.MemStats{},
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)

	agent.Worker(ctx, func() error {
		runtime.ReadMemStats(myM.Memstats)
		return nil
	}, time.Duration(cfg.PollInterval)*time.Second, 1)

	agent.Worker(ctx, func() error {
		agent.SendJSONData(cfg, agent.GetGopsutilMetrics())
		return nil
	}, time.Duration(cfg.PollInterval)*time.Second, cfg.RateLimit)

	agent.Worker(ctx, func() error {
		agent.SendMetric(cfg, myM.GetMetrics())
		agent.SendJSONData(cfg, myM.GetMetrics())
		return nil
	}, time.Duration(cfg.ReportInterval)*time.Second, cfg.RateLimit)

	wg.Wait()
}
