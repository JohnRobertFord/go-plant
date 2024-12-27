package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/compress"
	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/logger"
	"github.com/JohnRobertFord/go-plant/internal/server"
	"github.com/go-chi/chi/v5"
)

func MetricRouter(m *server.MemStorage) chi.Router {

	r := chi.NewRouter()
	r.Use(logger.Logging, compress.GzipMiddleware, server.Middleware)
	r.Get("/", m.GetAll)
	r.Get("/ping", m.Ping)
	r.Route("/update/", func(r chi.Router) {
		r.Post("/", m.WriteJSONMetric)
		r.Post("/{MT}/{M}/{V}", m.WriteMetric)
	})
	r.Route("/value/", func(r chi.Router) {
		r.Post("/", m.GetJSONMetric)
		r.Get("/{MT}/{M}", m.GetMetric)
	})

	return r
}

func main() {

	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("cant start server: %e", err)
	}

	mem := server.NewMemStorage(cfg)

	if cfg.Restore {
		mem.Read4File(cfg.FilePath)
	}

	httpServer := &http.Server{
		Addr:    cfg.Bind,
		Handler: MetricRouter(mem),
	}

	if cfg.StoreInterval > 0 {
		sleep := time.Duration(cfg.StoreInterval) * time.Second
		go func(m *server.MemStorage, t time.Duration) {
			for {
				<-time.After(t)
				server.Write2File(m)
			}
		}(mem, sleep)
	}

	go func() {
		log.Fatal(httpServer.ListenAndServe())
		httpServer.Shutdown(context.Background())
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	<-sigChan
	server.Write2File(mem)

}
