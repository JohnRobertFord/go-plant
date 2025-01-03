package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/server"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics/cache"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics/diskfile"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics/postgres"
)

func main() {

	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("can't init config: %e", err)
	}
	log.Print(cfg)

	var storage metrics.Storage

	if cfg.DatabaseDsn != "" {
		storage = postgres.NewPostgresStorage(cfg)
	} else {
		storage = cache.NewMemStorage(cfg)
	}

	if cfg.Restore {
		diskfile.Read4File(storage)
	}

	if cfg.StoreInterval > 0 && cfg.DatabaseDsn == "" {
		sleep := time.Duration(cfg.StoreInterval) * time.Second
		go func(ms metrics.Storage, t time.Duration) {
			for {
				<-time.After(t)
				diskfile.Write2File(ms)
				// fmt.Println("Write2File")
			}
		}(storage, sleep)
	}

	metricServer := server.NewMetricServer(cfg, storage)

	go metricServer.RunServer()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	<-sigChan
	diskfile.Write2File(storage)
}
