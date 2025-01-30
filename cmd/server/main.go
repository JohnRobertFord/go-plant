package main

import (
	"context"
	"fmt"
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

	var storage metrics.Storage
	ctx := context.Background()

	if cfg.DatabaseDsn != "" {
		storage, err = postgres.NewPostgresStorage(cfg)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		storage = cache.NewMemStorage(cfg)
	}

	if cfg.Restore {
		err := diskfile.Read4File(ctx, storage)
		if err != nil {
			log.Printf("[ERR][FILE] cant restore from file: %s", err)
		}
	}

	if cfg.StoreInterval > 0 && cfg.DatabaseDsn == "" {
		sleep := time.Duration(cfg.StoreInterval) * time.Second
		go func(ms metrics.Storage, t time.Duration) {
			for {
				<-time.After(t)
				err := diskfile.Write2File(ctx, ms)
				if err != nil {
					log.Printf("[ERR][FILE] cant write to file: %s", err)
				}
			}
		}(storage, sleep)
	}

	metricServer := server.NewMetricServer(cfg, storage)
	fmt.Println(cfg)
	go metricServer.RunServer()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	<-sigChan
	err = diskfile.Write2File(ctx, storage)
	if err != nil {
		log.Printf("[ERR][FILE] cant write to file: %s", err)
	}
}
