package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/compress"
	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/logger"
	"github.com/JohnRobertFord/go-plant/internal/server"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func MetricRouter(m *server.MemStorage) chi.Router {

	r := chi.NewRouter()
	r.Use(logger.Logging, compress.GzipMiddleware, server.Middleware)
	r.Use(middleware.SetHeader("Content-Type", "text/plain"))
	r.Get("/", m.GetAll)
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
		go func(m *server.MemStorage, t int) {
			ticker := time.NewTicker(time.Duration(t) * time.Second)
			for {
				select {
				case <-ticker.C:
					server.Write2File(m)
				}
			}
		}(mem, cfg.StoreInterval)
	}

	log.Fatal(httpServer.ListenAndServe())
	httpServer.Shutdown(context.Background())

}
